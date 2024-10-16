package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"vmware-automation/pkg/logging"
	"vmware-automation/pkg/vm"
)

func main() {
	// Переменные для настройки окружения
	vmrunPath := `C:\Program Files (x86)\VMware\VMware Workstation\vmrun.exe`
	sshKeyPath := `C:\Users\User\.ssh\id_rsa_no_pass`
	sshUser := "user"
	snapshotName := "Install ssh"

	// Определяем виртуальные машины
	vms := getVMs()

	// Команды для выполнения на каждой машине
	commands := []string{
		"sudo apt update",
		"sleep 10",
		"cd /mnt/hgfs/Shared/analytic4/docker/astra/1.7_x86-64 && sudo dpkg -i *.deb",
		"cd /mnt/hgfs/Shared/analytic4 && sudo bash install-docker.sh",
		"cd /mnt/hgfs/Shared/analytic4 && sudo bash install.sh",
	}

	// Цикл по виртуальным машинам
	for _, vmDetails := range vms {
		processVM(vmrunPath, sshUser, sshKeyPath, snapshotName, vmDetails, commands)
	}
}

// Удаляет старый файл лога, если он существует
func cleanupLogFile(logFilePath string) {
	if _, err := os.Stat(logFilePath); err == nil {
		if err = os.Remove(logFilePath); err != nil {
			log.Printf("Не удалось удалить файл %s: %v", logFilePath, err)
		} else {
			log.Printf("Файл %s удален.\n", logFilePath)
		}
	} else if !os.IsNotExist(err) {
		log.Printf("Ошибка проверки существования файла %s: %v", logFilePath, err)
	}
}

// Возвращает список виртуальных машин для тестирования
func getVMs() []struct {
	vmxPath string
	sshHost string
} {
	return []struct {
		vmxPath string
		sshHost string
	}{
		{`C:\Users\User\Documents\Virtual Machines\Astra-linux 1.7.5\Astra-linux 1.7.5.vmx`, "192.168.71.128"},
		// Добавьте другие виртуальные машины при необходимости
	}
}

// Обрабатывает одну виртуальную машину: откат к снапшоту, запуск, тестирование, логирование
func processVM(vmrunPath, sshUser, sshKeyPath, snapshotName string, vmDetails struct {
	vmxPath string
	sshHost string
}, commands []string) {
	// Откат к снапшоту
	vm.RevertToSnapshot(vmrunPath, vmDetails.vmxPath, snapshotName)

	// Запуск виртуальной машины
	if !vm.StartVM(vmrunPath, vmDetails.vmxPath) {
		vm.StopVM(vmrunPath, vmDetails.vmxPath)
		return // Пропускаем, если запуск не удался
	}

	// Извлечение версии сборки
	version := extractVersionFromPath(vmDetails.vmxPath)

	// Подключение по SSH и выполнение команд
	testPassed := vm.RunCommands(sshUser, vmDetails.sshHost, sshKeyPath, commands)

	// Проверка тестов
	if !testPassed {
		formattedMessageTests := fmt.Sprintf("Тест не пройден на версии сборки %s.\n", version)
		logging.WriteLogToFile(formattedMessageTests)
		vm.StopVM(vmrunPath, vmDetails.vmxPath)
		return
	}

	// Отправка curl-запросов через SSH
	vm.SendCurlViaSSH(sshUser, vmDetails.sshHost, sshKeyPath)
	formattedMessageCurl := fmt.Sprintf("Все команды выполнены успешно. Тест пройден на версии сборки %s.\n", version)
	logging.WriteLogToFile(formattedMessageCurl)
	//log.Printf("Все команды выполнены успешно. Тест пройден на версии сборки %s.\n", version)

	// Остановка виртуальной машины
	vm.StopVM(vmrunPath, vmDetails.vmxPath)
}

// Извлекает версию сборки из пути к VMX-файлу
func extractVersionFromPath(vmxPath string) string {
	base := filepath.Base(vmxPath)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
