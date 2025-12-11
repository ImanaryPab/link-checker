package pdf

import (
	"fmt"
	"time"

	"link-checker/internal/storage"

	"github.com/jung-kurt/gofpdf"
)

func GenerateReport(tasks []*storage.Task) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Заголовок
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(190, 10, "Отчет по проверке ссылок")
	pdf.Ln(15)

	// Для каждой задачи
	for _, task := range tasks {
		pdf.SetFont("Arial", "B", 12)
		pdf.Cell(190, 8, fmt.Sprintf("Задача #%d - %s",
			task.ID,
			task.CreatedAt.Format("2006-01-02 15:04:05")))
		pdf.Ln(8)

		pdf.SetFont("Arial", "", 10)

		// Заголовки таблицы
		pdf.SetFont("Arial", "B", 10)
		pdf.Cell(120, 8, "Ссылка")
		pdf.Cell(70, 8, "Статус")
		pdf.Ln(8)
		pdf.SetFont("Arial", "", 10)

		// Данные
		row := 0
		for link, status := range task.Links {
			// Чередование цвета фона строк
			if row%2 == 0 {
				pdf.SetFillColor(255, 255, 255)
			} else {
				pdf.SetFillColor(245, 245, 245)
			}

			// Статус
			statusText := ""
			switch status {
			case storage.StatusAvailable:
				statusText = "✓ Доступен"
				pdf.SetTextColor(0, 128, 0) // Зеленый
			case storage.StatusUnavailable:
				statusText = "✗ Не доступен"
				pdf.SetTextColor(255, 0, 0) // Красный
			case storage.StatusProcessing:
				statusText = "⏳ В обработке"
				pdf.SetTextColor(255, 165, 0) // Оранжевый
			default:
				statusText = "⚠️ Ошибка"
				pdf.SetTextColor(128, 128, 128) // Серый
			}

			// Ссылка
			pdf.SetTextColor(0, 0, 0)
			pdf.CellFormat(120, 8, truncateString(link, 60), "1", 0, "L", true, 0, "")

			// Статус
			pdf.SetTextColor(0, 0, 0)
			pdf.CellFormat(70, 8, statusText, "1", 0, "L", true, 0, "")
			pdf.Ln(8)

			row++
		}

		pdf.Ln(10)
	}

	// Итог
	pdf.SetFont("Arial", "I", 10)
	pdf.Cell(190, 8, fmt.Sprintf("Всего задач: %d | Сгенерировано: %s",
		len(tasks),
		time.Now().Format("2006-01-02 15:04:05")))
	pdf.Ln(15)

	return pdf.OutputBytes()
}

func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}
