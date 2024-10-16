package vm

import (
	"fmt"
	"log"
	"os"
	"strings"
	"vmware-automation/pkg/logging"

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

	// Проверка на наличие HTML или ошибки 502
	if strings.Contains(formattedOutput, "<html>") || strings.Contains(formattedOutput, "502 Bad Gateway") {
		log.Println("Ошибка: сервер вернул HTML-код или 502 Bad Gateway вместо ожидаемого JSON.")
		return
	}

	// Проверка на правильный JSON-ответ
	if !strings.Contains(formattedOutput, `"userId":"admin"`) {
		log.Println("Ошибка: неправильный формат JSON-ответа.")
		return
	}

	log.Println(formattedOutput)

	// Логируем вывод в файл
	logging.WriteLogToFile(formattedOutput)
}

// Функция для форматирования вывода curl
func formatCurlOutput(output string) string {
	lines := strings.Split(output, "\n")
	var statsLines []string
	var responseLines []string
	foundJSON := false

	// Проходим по строкам и сортируем их по типу (статистика/ответ)
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "{") || foundJSON {
			// Если начинается JSON или мы уже начали его собирать
			if !foundJSON && len(statsLines) > 0 {
				// Добавляем пустую строку перед JSON-ответом
				statsLines = append(statsLines, "")
			}
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
