package vm

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"vmware-automation/pkg/logging"

	"golang.org/x/crypto/ssh"
)

// Функция отправки HTTP-запросов
func sendCurl(step string) {
	_, err := http.Get("http://localhost:8080/track?step=" + step)
	if err != nil {
		log.Printf("Не удалось отправить запрос: %v\n", err)
	}
}

// Откат к Snapshot
func RevertToSnapshot(vmrunPath, vmxPath, snapshotName string) bool {
	sendCurl("revert_snapshot")
	cmd := exec.Command(vmrunPath, "revertToSnapshot", vmxPath, snapshotName)
	if err := cmd.Run(); err != nil {
		log.Printf("Ошибка при откате к Snapshot: %v\n", err)
		StopVM(vmrunPath, vmxPath) // Останавливаем виртуальную машину при ошибке
		return false
	}
	log.Println("Откат к Snapshot успешно выполнен")
	return true
}

// Запуск виртуальной машины
func StartVM(vmrunPath, vmxPath string) bool {
	sendCurl("start_vm")
	cmd := exec.Command(vmrunPath, "start", vmxPath, "gui")
	if err := cmd.Run(); err != nil {
		log.Printf("Ошибка при запуске виртуальной машины: %v\n", err)
		StopVM(vmrunPath, vmrunPath)
		return false
	}
	log.Println("Виртуальная машина успешно запущена")
	time.Sleep(10 * time.Second)
	return true
}

// Остановка виртуальной машины
func StopVM(vmrunPath, vmxPath string) {
	cmd := exec.Command(vmrunPath, "stop", vmxPath, "soft")
	if err := cmd.Run(); err != nil {
		log.Printf("Ошибка при выключении виртуальной машины: %v\n", err)
	}
	log.Println("Виртуальная машина успешно выключена.")
}

// Функция для парсинга результатов sudo docker info
func parseDockerInfo(output string) map[string]int {
	result := make(map[string]int)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Containers:") || strings.Contains(line, "Running:") || strings.Contains(line, "Paused:") || strings.Contains(line, "Stopped:") {
			parts := strings.Split(strings.TrimSpace(line), ": ")
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value, err := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err == nil {
					result[key] = value
				}
			}
		}
	}
	return result
}

// Проверка статуса контейнеров
func checkDockerContainers(client *ssh.Client) (bool, map[string]int) {
	session, err := client.NewSession()
	if err != nil {
		log.Printf("Не удалось создать сессию для проверки Docker: %v\n", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput("sudo docker info")
	if err != nil {
		log.Printf("Ошибка при выполнении команды docker info: %v\n", err)
	}

	dockerInfo := parseDockerInfo(string(output))
	logMessage := fmt.Sprintf("Проверка статуса контейнеров:\nContainers: %d\nRunning: %d\nPaused: %d\nStopped: %d\n",
		dockerInfo["Containers"], dockerInfo["Running"], dockerInfo["Paused"], dockerInfo["Stopped"])
	log.Println(logMessage)
	logging.WriteLogToFile(logMessage)

	// Проверка: все контейнеры должны быть запущены
	if dockerInfo["Running"] == 16 && dockerInfo["Stopped"] == 0 {
		return true, dockerInfo
	}

	return false, dockerInfo
}

// Подключение по SSH и выполнение команд
func RunCommands(sshUser, sshHost, sshKeyPath string, commands []string) bool {
	// Удаляем лог-файл при старте
	logging.DeleteLogFile()

	sendCurl("ssh_connect")
	key, err := os.ReadFile(sshKeyPath)
	if err != nil {
		log.Printf("Не удалось прочитать SSH ключ: %v\n", err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Printf("Не удалось распарсить ключ SSH: %v\n", err)
	}
	config := &ssh.ClientConfig{
		User:            sshUser,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	sshAddress := fmt.Sprintf("%s:%d", sshHost, 22)
	client, err := ssh.Dial("tcp", sshAddress, config)
	if err != nil {
		log.Printf("Не удалось подключиться к %s: %v\n", sshAddress, err)
	}
	defer client.Close()

	// Выполнение install-docker.sh
	for _, cmd := range commands {
		log.Printf("Выполняется команда: %s\n", cmd)
		session, err := client.NewSession()
		if err != nil {
			log.Printf("Не удалось создать сессию: %v\n", err)
			StopVM(sshHost, sshHost)
			return false
		}

		// Вывод команды в реальном времени
		session.Stdout = os.Stdout
		session.Stderr = os.Stderr

		err = session.Run(cmd) // Выполняем команду
		if err != nil {
			log.Printf("Команда завершилась с ошибкой: %v\n", err)
			session.Close()
			StopVM(sshHost, sshHost)
			return false
		}

		log.Printf("Команда завершена: %s\n", cmd)
		session.Close()
	}

	// Проверка статуса контейнеров и повторная попытка запуска install.sh при необходимости
	attempt := 1
	for attempt <= 2 { //2 || 1
		time.Sleep(10 * time.Second) // Задержка перед проверкой

		// Проверка статуса контейнеров
		allRunning, dockerInfo := checkDockerContainers(client)
		if allRunning {
			log.Println("Все контейнеры запущены успешно.")
			logging.WriteLogToFile("Все контейнеры запущены успешно.")
			return true
		}

		// Если есть остановленные контейнеры, перезапуск install.sh
		if dockerInfo["Stopped"] > 0 {
			log.Printf("Обнаружено %d остановленных контейнеров. Попытка %d перезапуска install.sh.\n", dockerInfo["Stopped"], attempt)
			logging.WriteLogToFile(fmt.Sprintf("Обнаружено %d остановленных контейнеров. Попытка %d перезапуска install.sh.\n", dockerInfo["Stopped"], attempt))
			session, err := client.NewSession()
			if err != nil {
				log.Printf("Не удалось создать сессию для перезапуска install.sh: %v\n", err)
			}

			session.Stdout = os.Stdout
			session.Stderr = os.Stderr
			err = session.Run("cd /mnt/hgfs/Shared/analytic4 && sudo bash install.sh")
			if err != nil {
				log.Printf("Перезапуск install.sh завершился с ошибкой: %v\n", err)
				session.Close()
				StopVM(sshHost, sshHost)
				return false
			}

			session.Close()
			log.Println("Перезапуск install.sh завершен.")
			logging.WriteLogToFile("Перезапуск install.sh завершен.")
		}

		attempt++
	}

	// Если после двух попыток контейнеры не запущены, возвращаем false
	allRunning, _ := checkDockerContainers(client)
	if !allRunning {
		log.Println("Один или несколько контейнеров находятся в статусе Stopped. Тест не пройден.")
		logging.WriteLogToFile("Один или несколько контейнеров находятся в статусе Stopped. Тест не пройден.")
		return false
	}

	return true
}
