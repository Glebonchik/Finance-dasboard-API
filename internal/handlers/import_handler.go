package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gibbon/finace-dashboard/internal/dto"
	"github.com/gibbon/finace-dashboard/internal/middleware"
	"github.com/gibbon/finace-dashboard/internal/parser"
	"github.com/gibbon/finace-dashboard/internal/tasks"
	"github.com/hibiken/asynq"
)

type ImportHandler struct {
	asynqClient *asynq.Client
}


func NewImportHandler(client *asynq.Client) *ImportHandler {
	return &ImportHandler{
		asynqClient: client,
	}
}

// Import
// @Summary Импорт транзакций из файла
// @Description Загрузка CSV или Excel файла с транзакциями
// @Tags imports
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Файл для импорта (CSV или Excel)"
// @Param format formData string true "Формат файла (sberbank_csv, tinkoff_csv, excel)"
// @Success 200 {object} dto.ImportResponse
// @Failure 400 {object} map[string]string "Некорректные данные"
// @Failure 401 {object} map[string]string "Неавторизован"
// @Failure 500 {object} map[string]string "Ошибка сервера"
// @Router /api/v1/imports [post]
func (h *ImportHandler) Import(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Ограничение на размер файла: 10MB
	r.ParseMultipartForm(10 << 20)

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"error": "file is required"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	format := r.FormValue("format")
	if format == "" {
		http.Error(w, `{"error": "format is required"}`, http.StatusBadRequest)
		return
	}

	fileData, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, `{"error": "failed to read file"}`, http.StatusInternalServerError)
		return
	}

	payload := tasks.ImportTransactionsPayload{
		UserID:   userID,
		Format:   format,
		FileData: fileData,
	}

	task, err := tasks.NewImportTransactionsTask(payload)
	if err != nil {
		http.Error(w, `{"error": "failed to create import task"}`, http.StatusInternalServerError)
		return
	}

	info, err := h.asynqClient.Enqueue(task, asynq.Queue("default"), asynq.MaxRetry(3))
	if err != nil {
		http.Error(w, `{"error": "failed to enqueue task"}`, http.StatusInternalServerError)
		return
	}


	_ = info

	response := dto.ImportResponse{
		TotalRows: 0, // Будет известно после обработки
		SuccessCount: 0,
		FailedCount:  0,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted) // 202 Accepted
	json.NewEncoder(w).Encode(response)
}

// ImportSync
// @Summary Синхронный импорт транзакций
// @Description Загрузка CSV или Excel файла с синхронной обработкой
// @Tags imports
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Файл для импорта (CSV или Excel)"
// @Param format formData string true "Формат файла (sberbank_csv, tinkoff_csv, excel)"
// @Success 200 {object} dto.ImportResponse
// @Failure 400 {object} map[string]string "Некорректные данные"
// @Failure 401 {object} map[string]string "Неавторизован"
// @Failure 500 {object} map[string]string "Ошибка сервера"
// @Router /api/v1/imports/sync [post]
func (h *ImportHandler) ImportSync(w http.ResponseWriter, r *http.Request) {
	_, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Ограничение на размер файла: 10MB
	r.ParseMultipartForm(10 << 20)

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"error": "file is required"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	format := r.FormValue("format")
	if format == "" {
		http.Error(w, `{"error": "format is required"}`, http.StatusBadRequest)
		return
	}

	var p parser.Parser
	switch format {
	case "sberbank_csv":
		p = parser.NewSberbankCSVParser()
	case "tinkoff_csv":
		p = parser.NewTinkoffCSVParser()
	case "excel":
		p = parser.NewExcelParser()
	default:
		http.Error(w, `{"error": "unsupported format"}`, http.StatusBadRequest)
		return
	}

	transactions, err := p.Parse(file)
	if err != nil {
		http.Error(w, `{"error": "failed to parse file: `+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	response := dto.ImportResponse{
		TotalRows:    len(transactions),
		SuccessCount: len(transactions),
		FailedCount:  0,
		Transactions: make([]dto.ImportedTransaction, 0, len(transactions)),
	}

	for _, tx := range transactions {
		response.Transactions = append(response.Transactions, dto.ImportedTransaction{
			Description: tx.Description,
			Amount:      tx.Amount,
			Currency:    tx.Currency,
			Date:        tx.Date.Format("2006-01-02"),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetImportStatus
// @Summary Получить статус импорта
// @Description Получение статуса задачи импорта по task_id
// @Tags imports
// @Produce json
// @Param task_id query string true "ID задачи импорта"
// @Success 200 {object} dto.ImportResponse
// @Failure 400 {object} map[string]string "Некорректные данные"
// @Router /api/v1/imports/status [get]
func (h *ImportHandler) GetImportStatus(w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		http.Error(w, `{"error": "task_id is required"}`, http.StatusBadRequest)
		return
	}

	// TODO: Получить статус задачи из asynq
	// Пока заглушка
	response := dto.ImportResponse{
		TotalRows:    0,
		SuccessCount: 0,
		FailedCount:  0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
