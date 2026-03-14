package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

const (
	TypeImportTransactions = "import:transactions"
)


type ImportTransactionsPayload struct {
	UserID   string `json:"user_id"`
	Format   string `json:"format"`
	FileData []byte `json:"file_data"`
}


func NewImportTransactionsTask(payload ImportTransactionsPayload) (*asynq.Task, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}
	return asynq.NewTask(TypeImportTransactions, b), nil
}

type ImportHandler struct {
	// Здесь будут зависимости (сервисы)
}

func NewImportHandler() *ImportHandler {
	return &ImportHandler{}
}

func (h *ImportHandler) ProcessImportTask(ctx context.Context, t *asynq.Task) error {
	var p ImportTransactionsPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// TODO: Здесь будет логика обработки файла
	// 1. Распарсить файл в зависимости от формата
	// 2. Валидировать данные
	// 3. Создать транзакции через TransactionService
	// 4. Применить автоматическую категоризацию

	fmt.Printf("Processing import task for user %s, format: %s\n", p.UserID, p.Format)

	return nil
}
