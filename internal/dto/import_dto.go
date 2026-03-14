package dto

type ImportTransactionRequest struct {
	Format string `json:"format"` // "sberbank_csv", "tinkoff_csv", "excel"
}

type ImportResponse struct {
	TotalRows     int `json:"total_rows"`
	SuccessCount  int `json:"success_count"`
	FailedCount   int `json:"failed_count"`
	Transactions  []ImportedTransaction `json:"transactions,omitempty"`
	Errors        []ImportError `json:"errors,omitempty"`
}

type ImportedTransaction struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Amount      float64 `json:"amount"`
	Currency    string `json:"currency"`
	Date        string `json:"date"`
	CategoryID  *int `json:"category_id,omitempty"`
}

type ImportError struct {
	RowNumber int `json:"row_number"`
	RawData   string `json:"raw_data"`
	Error     string `json:"error"`
}
