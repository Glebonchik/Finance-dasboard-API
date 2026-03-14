package parser

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

type ExcelParser struct{}

func NewExcelParser() *ExcelParser {
	return &ExcelParser{}
}

func (p *ExcelParser) Parse(reader io.Reader) ([]ParsedTransaction, error) {
	f, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read rows: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("file is empty or has no data rows")
	}

	var transactions []ParsedTransaction

	for i, row := range rows[1:] {
		lineNum := i + 2 // +2 потому что 0-based индекс + пропущенный заголовок

		if len(row) < 4 {
			return nil, fmt.Errorf("line %d: insufficient columns", lineNum)
		}

		// Ожидаемый формат:
		// 0: Дата, 1: Сумма, 2: Валюта, 3: Описание
		dateStr := row[0]
		date, err := parseExcelDate(dateStr)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid date: %w", lineNum, err)
		}

		amount, err := parseExcelAmount(row[1])
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid amount: %w", lineNum, err)
		}

		currency := row[2]
		description := row[3]

		transactions = append(transactions, ParsedTransaction{
			Date:        date,
			Amount:      amount,
			Currency:    currency,
			Description: description,
		})
	}

	return transactions, nil
}


func parseExcelDate(dateStr string) (time.Time, error) {
	// Пробуем разные форматы
	formats := []string{
		"02.01.2006",
		"02/01/2006",
		"2006-01-02",
		"02.01.2006 15:04",
		"02/01/2006 15:04",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unrecognized date format: %s", dateStr)
}

func parseExcelAmount(amountStr string) (float64, error) {
	// Заменяем запятую на точку для совместимости
	amountStr = strings.ReplaceAll(amountStr, ",", ".")
	return strconv.ParseFloat(strings.TrimSpace(amountStr), 64)
}
