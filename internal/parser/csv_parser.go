package parser

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

type ParsedTransaction struct {
	Date        time.Time
	Amount      float64
	Currency    string
	Description string
}

type Parser interface {
	Parse(reader io.Reader) ([]ParsedTransaction, error)
}


type SberbankCSVParser struct{}

func NewSberbankCSVParser() *SberbankCSVParser {
	return &SberbankCSVParser{}
}

func (p *SberbankCSVParser) Parse(reader io.Reader) ([]ParsedTransaction, error) {
	csvReader := csv.NewReader(reader)
	csvReader.Comma = ';'
	csvReader.LazyQuotes = true

	// Пропускаем заголовок
	_, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	var transactions []ParsedTransaction
	lineNum := 1

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		lineNum++

		// Формат Сбербанка:
		// 0: Дата операции, 1: Тип операции, 2: Сумма, 3: Валюта, 4: Описание
		if len(record) < 5 {
			return nil, fmt.Errorf("line %d: insufficient columns", lineNum)
		}

		// Парсим дату (формат: 02.01.2024 10:30)
		dateStr := strings.TrimSpace(record[0])
		date, err := time.Parse("02.01.2006 15:04", dateStr)
		if err != nil {
			// Пробуем без времени
			date, err = time.Parse("02.01.2006", dateStr)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid date format: %w", lineNum, err)
			}
		}

		// Парсим сумму
		amountStr := strings.ReplaceAll(strings.TrimSpace(record[2]), ",", ".")
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid amount: %w", lineNum, err)
		}

		currency := strings.TrimSpace(record[3])
		description := strings.TrimSpace(record[4])

		transactions = append(transactions, ParsedTransaction{
			Date:        date,
			Amount:      amount,
			Currency:    currency,
			Description: description,
		})
	}

	return transactions, nil
}


type TinkoffCSVParser struct{}

func NewTinkoffCSVParser() *TinkoffCSVParser {
	return &TinkoffCSVParser{}
}

func (p *TinkoffCSVParser) Parse(reader io.Reader) ([]ParsedTransaction, error) {
	csvReader := csv.NewReader(reader)
	csvReader.Comma = ','
	csvReader.LazyQuotes = true

	// Пропускаем заголовок
	_, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	var transactions []ParsedTransaction
	lineNum := 1

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		lineNum++

		// Формат Тинькофф:
		// 0: Дата, 1: Сумма, 2: Валюта, 3: Описание, 4: Категория
		if len(record) < 4 {
			return nil, fmt.Errorf("line %d: insufficient columns", lineNum)
		}

		// Парсим дату (формат: 01.02.2024)
		dateStr := strings.TrimSpace(record[0])
		date, err := time.Parse("02.01.2006", dateStr)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid date format: %w", lineNum, err)
		}

		// Парсим сумму
		amountStr := strings.ReplaceAll(strings.TrimSpace(record[1]), ",", ".")
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid amount: %w", lineNum, err)
		}

		currency := strings.TrimSpace(record[2])
		description := strings.TrimSpace(record[3])

		transactions = append(transactions, ParsedTransaction{
			Date:        date,
			Amount:      amount,
			Currency:    currency,
			Description: description,
		})
	}

	return transactions, nil
}
