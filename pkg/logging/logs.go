package logging

import (
	"fmt"
	"log"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Логирование результатов тестирования в xlsx
func LogResultsToXLSX(version, installDockerStatus, installStatus, containerStatus, curlStatus string, retries int, filepath string) error {
	// Открываем существующий файл или создаем новый, если его нет
	f, err := excelize.OpenFile(filepath)
	if err != nil {
		f = excelize.NewFile()
		// Подписи заголовков, если создается новый файл
		sheet := "Результаты тестов"
		f.NewSheet(sheet)
		headers := []string{"Версия Astra", "Установка install-docker.sh", "Установка install.sh", "Состояние контейнеров", "Количество перезапусков install.sh", "Ответ от сервера (статус curl)"}
		for i, header := range headers {
			col := fmt.Sprintf("%c1", 'A'+i)
			f.SetCellValue(sheet, col, header)
		}
	}

	// Поиск последней строки с данными
	sheet := "Результаты тестов"
	rows, err := f.GetRows(sheet)
	if err != nil {
		return fmt.Errorf("Ошибка чтения строк: %v", err)
	}
	rowNum := len(rows) + 1

	// Заполнение данных
	f.SetCellValue(sheet, fmt.Sprintf("A%d", rowNum), version)
	f.SetCellValue(sheet, fmt.Sprintf("B%d", rowNum), installDockerStatus)
	f.SetCellValue(sheet, fmt.Sprintf("C%d", rowNum), installStatus)
	f.SetCellValue(sheet, fmt.Sprintf("D%d", rowNum), containerStatus)
	f.SetCellValue(sheet, fmt.Sprintf("E%d", rowNum), retries)
	f.SetCellValue(sheet, fmt.Sprintf("F%d", rowNum), curlStatus)

	// Сохранение файла
	if err := f.SaveAs(filepath); err != nil {
		return fmt.Errorf("не удалось сохранить файл: %v", err)
	}

	log.Printf("Результаты записаны в файл: %s\n", filepath)
	return nil
}

// Проверка состояния контейнеров с приоритетом ошибок
func CheckContainerStatus(output string, err error) string {
	// Приоритет ошибки контейнеров
	if strings.Contains(output, "Exited") || strings.Contains(output, "Stopped") {
		return "Ошибка: контейнеры остановлены"
	}

	// Если с контейнерами всё в порядке, проверяем другие ошибки
	if err != nil {
		return fmt.Sprintf("Ошибка: %v", err)
	}

	// Если ошибок нет, возвращаем "ОК"
	return "ОК"
}

// Форматирование статуса установки с приоритетом ошибки
func FormatInstallStatus(output string, err error) string {
	// Если возникла ошибка, возвращаем её текст
	if err != nil {
		return fmt.Sprintf("Ошибка: %v", err)
	}

	// Если в выводе есть ошибка (например, остановка процесса), возвращаем её
	if strings.Contains(output, "Failed") || strings.Contains(output, "Error") {
		return "Ошибка: установка не удалась"
	}

	// Если ошибок нет, возвращаем "ОК"
	return "ОК"
}

// Форматирование статуса curl
func FormatCurlStatus(output string) string {
	if strings.Contains(output, "502 Bad Gateway") || strings.Contains(output, "<html>") {
		return "Ошибка: неправильный ответ от сервера"
	}
	return "ОК"
}
