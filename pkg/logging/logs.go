package logging

import (
	"log"
	"os"
)

// Функция для записи логов в файл
func WriteLogToFile(logMessage string) {
	file, err := os.OpenFile("vm_log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Не удалось открыть файл для записи логов: %v\n", err)
		return
	}
	defer file.Close()

	if _, err := file.WriteString(logMessage + "\n"); err != nil {
		log.Printf("Не удалось записать лог в файл: %v\n", err)
	}
}

// Функция для удаления файла логов
func DeleteLogFile() {
	if _, err := os.Stat("vm_log.txt"); err == nil {
		// Файл существует, удаляем его
		err = os.Remove("vm_log.txt")
		if err != nil {
			log.Printf("Не удалось удалить файл логов: %v\n", err)
		} else {
			log.Println("Файл логов успешно удален.")
		}
	}
}
