package main

import (
	"log"

	"vmware-automation/pkg/vm"
)

func main() {
	// Переменные
	vmrunPath := `C:\Program Files (x86)\VMware\VMware Workstation\vmrun.exe`
	vmxPath := `C:\Users\User\Documents\Virtual Machines\Astra-linux 1.7.5\Astra-linux 1.7.5.vmx`
	sshKeyPath := `C:\Users\User\.ssh\id_rsa_no_pass`
	sshUser := "user"
	sshHost := "192.168.71.128"
	snapshotName := "Install ssh"

	// Откат к снапшоту
	vm.RevertToSnapshot(vmrunPath, vmxPath, snapshotName)

	// Запуск виртуальной машины
	vm.StartVM(vmrunPath, vmxPath)

	// Подключение по SSH и выполнение команд
	commands := []string{
		"sudo apt update",
		"sleep 10",
		"cd /mnt/hgfs/Shared/analytic4/docker/astra/1.7_x86-64 && sudo dpkg -i *.deb", //Astra-linux || Shared
		"cd /mnt/hgfs/Shared/analytic4 && sudo bash install-docker.sh",
		"cd /mnt/hgfs/Shared/analytic4 && sudo bash install.sh",
		//"sudo bash install.sh",
	}
	testPassed := vm.RunCommands(sshUser, sshHost, sshKeyPath, commands)

	if !testPassed {
		log.Println("Тест не пройден: один или несколько контейнеров находятся в статусе Stopped.")
		return
	}

	// Отправка curl через SSH для авторизации только если тест пройден
	vm.SendCurlViaSSH(sshUser, sshHost, sshKeyPath)

	// Отправка POST запроса на авторизацию только если тест пройден
	// url := "http://localhost/api/auth/v1/login"
	// login := "admin"
	// password := "admin"
	// vm.SendPostRequest(url, login, password)

	// Лог завершения
	log.Println("Все команды выполнены успешно. Тест пройден.")
}
