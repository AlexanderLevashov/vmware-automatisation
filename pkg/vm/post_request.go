package vm

import (
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

// Отправка curl через SSH на виртуальную машину
func SendCurlViaSSH(sshUser, sshHost, sshKeyPath string) {
	key, err := os.ReadFile(sshKeyPath)
	if err != nil {
		log.Fatalf("Не удалось прочитать SSH ключ: %v\n", err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatalf("Не удалось распарсить ключ SSH: %v\n", err)
	}
	config := &ssh.ClientConfig{
		User:            sshUser,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	sshAddress := fmt.Sprintf("%s:%d", sshHost, 22)
	client, err := ssh.Dial("tcp", sshAddress, config)
	if err != nil {
		log.Fatalf("Не удалось подключиться к %s: %v\n", sshAddress, err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		log.Fatalf("Не удалось создать сессию: %v\n", err)
	}
	defer session.Close()

	// Выполнение команды curl на виртуальной машине
	cmd := `curl -X POST http://localhost/api/auth/v1/login -d '{"login":"admin", "password":"admin"}' -H "Content-Type: application/json"`
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		log.Fatalf("Ошибка при выполнении команды curl: %v\n", err)
	}

	// Форматируем вывод
	formattedOutput := formatCurlOutput(string(output))

	//log.Printf("Ответ от curl: %s\n", string(output))
	log.Println(formattedOutput)
}

// Функция для форматирования вывода curl
func formatCurlOutput(output string) string {
	lines := strings.Split(output, "\n")
	var statsLines []string
	var responseLines []string
	foundJSON := false

	// Проходим по строкам и сортируем их по типу (статистика/ответ)
	for _, line := range lines {
		if strings.HasPrefix(line, "{") || foundJSON {
			// Если начинается JSON или мы уже начали его собирать
			responseLines = append(responseLines, line)
			foundJSON = true
		} else {
			// Остальные строки считаем статистикой
			statsLines = append(statsLines, line)
		}
	}

	// Собираем все в нужном формате
	statsBlock := strings.Join(statsLines, "\n")
	responseBlock := strings.Join(responseLines, "\n")

	return fmt.Sprintf("\nСтатистика загрузки:\n%s\n\nОтвет от сервера:\n%s", statsBlock, responseBlock)
}
