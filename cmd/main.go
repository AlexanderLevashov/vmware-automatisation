package main

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"vmware-automation/pkg/logging"
	"vmware-automation/pkg/vm"
)

func main() {
	logFilePath := "test_results.xlsx" // Определяем путь для xlsx файла
	// Переменные
	vmrunPath := `C:\Program Files (x86)\VMware\VMware Workstation\vmrun.exe`
	sshKeyPath := `C:\Users\User\.ssh\id_rsa_no_pass`
	sshUser := "user"
	snapshotName := "Install ssh"

	vms := []struct {
		vmxPath string
		sshHost string
	}{
		{`C:\Users\User\Documents\Virtual Machines\Astra-linux 1.7.5\Astra-linux 1.7.5.vmx`, "192.168.71.128"},
		//{`D:\Documents\Virtual Machines\Astra-linux Installation-1.7.5.9-16.10.23\Astra-linux Installation-1.7.5.9-16.10.23.vmx`, "192.168.139.139"},
		//{`D:\Documents\Virtual Machines\Astra-linux Installation-1.7.5.16-06.02.24\Astra-linux Installation-1.7.5.16-06.02.24.vmx`, "192.168.139.140"},
		//{`D:\Documents\Virtual Machines\Astra-linux 1.7.6.11-26.08.24\Astra-linux 1.7.6.11-26.08.24.vmx`, "192.168.139.141"},
		//{`D:\Documents\Virtual Machines\Astra-linux Installation-1.7.6.11-26.08.24\Astra-linux Installation-1.7.6.11-26.08.24.vmx`, "192.168.139.142"},
	}

	// Подключение по SSH и выполнение команд
	commands := []string{
		"sudo apt update",
		"sleep 10",
		"cd /mnt/hgfs/Shared/analytic4/docker/astra/1.7_x86-64 && sudo dpkg -i *.deb", //Astra-linux || Shared
		//"cd /mnt/hgfs/Shared/analytic4 && sudo bash install-docker.sh",
		"cd /mnt/hgfs/Shared/analytic4 && sudo bash install.sh",
	}
	// Цикл по виртуальным машинам
	for _, vmDetails := range vms {
		// Откат к снапшоту
		vm.RevertToSnapshot(vmrunPath, vmDetails.vmxPath, snapshotName)

		if !vm.StartVM(vmrunPath, vmDetails.vmxPath) {
			vm.StopVM(vmrunPath, vmDetails.vmxPath)
			continue // Переходим к следующей машине, если ошибка
		}

		// Запуск виртуальной машины
		//vm.StartVM(vmrunPath, vmDetails.vmxPath)

		// Извлечение версии сборки из пути к vmx-файлу
		version := extractVersionFromPath(vmDetails.vmxPath)

		// Подключение по SSH и выполнение команд
		testPassed, err := vm.RunCommands(sshUser, vmDetails.sshHost, sshKeyPath, commands)

		outputInstall := "Installation Failed"
		installDockerStatus := logging.FormatInstallStatus(outputInstall, nil) // Команда прошла успешно
		installStatus := logging.FormatInstallStatus(outputInstall, err)
		fmt.Println(installStatus)

		outputInstallDocker := "Stopped"
		containerStatus := logging.CheckContainerStatus(outputInstallDocker, err) // Проверяем состояние контейнеров
		fmt.Println(containerStatus)

		curlStatus := logging.FormatCurlStatus("JSON response") // Проверка ответа от сервера

		// Записываем результаты в xlsx файл
		err = logging.LogResultsToXLSX(version, installDockerStatus, installStatus, containerStatus, curlStatus, 0, logFilePath)
		if err != nil {
			log.Printf("Ошибка записи логов: %v", err)
		}

		// Если тест не пройден или возникла ошибка
		if !testPassed {
			log.Printf("Тест не пройден на версии сборки %s.\n", version)
			vm.StopVM(vmrunPath, vmDetails.vmxPath) // Остановка виртуальной машины
			continue                                // Переход к следующей машине
		}

		// Отправка curl через SSH для авторизации, если тест пройден
		vm.SendCurlViaSSH(sshUser, vmDetails.sshHost, sshKeyPath)
		log.Printf("Все команды выполнены успешно. Тест пройден на версии сборки %s.\n", version)

		// Выключение виртуальной машины
		vm.StopVM(vmrunPath, vmDetails.vmxPath)
	}
}

// Функция для извлечения версии сборки из пути к VMX-файлу
func extractVersionFromPath(vmxPath string) string {
	// Получаем базовое имя файла, удаляем расширение
	base := filepath.Base(vmxPath)
	// Извлекаем версию из имени каталога
	version := strings.TrimSuffix(base, filepath.Ext(base))
	return version
}
