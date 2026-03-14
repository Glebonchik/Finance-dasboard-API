package service

import (
	"context"
	"strings"
	"time"

	"github.com/gibbon/finace-dashboard/internal/domain/model"
	"github.com/gibbon/finace-dashboard/internal/domain/repository"
	"github.com/gibbon/finace-dashboard/internal/grpc_client"
	"github.com/google/uuid"
)

type TransactionService interface {
	// Создаёт новую транзакцию с автоматической категоризацией
	Create(ctx context.Context, userID string, tx *model.Transaction) (*model.Transaction, error)

	// Возвращает транзакцию по ID
	GetByID(ctx context.Context, userID, id string) (*model.Transaction, error)

	// Возвращает транзакции пользователя с фильтрацией
	GetByUserID(ctx context.Context, filter model.TransactionFilter) ([]*model.Transaction, error)

	// Обновляет транзакцию
	Update(ctx context.Context, userID string, tx *model.Transaction) (*model.Transaction, error)

	// Удаляет транзакцию
	Delete(ctx context.Context, userID, id string) error

	// Выполняет категоризацию транзакции
	Categorize(ctx context.Context, userID string, tx *model.Transaction) error

	// Создаёт правило категоризации
	CreateRule(ctx context.Context, userID string, keyword string, categoryID int) (*model.UserCategoryRule, error)

	// Возвращает правила пользователя
	GetRules(ctx context.Context, userID string) ([]*model.UserCategoryRule, error)

	// Удаляет правило
	DeleteRule(ctx context.Context, userID, ruleID string) error

	// Возвращает все категории
	GetCategories(ctx context.Context) ([]*model.Category, error)
}

// Результат категоризации
type categorizationResult struct {
	categoryID  *int
	isConfirmed bool
}

type transactionServiceImpl struct {
	txRepo       repository.TransactionRepository
	categoryRepo repository.CategoryRepository
	ruleRepo     repository.UserCategoryRuleRepository
	mlClient     *grpc_client.MLClient
}

func NewTransactionService(
	txRepo repository.TransactionRepository,
	categoryRepo repository.CategoryRepository,
	ruleRepo repository.UserCategoryRuleRepository,
	mlClient *grpc_client.MLClient,
) TransactionService {
	return &transactionServiceImpl{
		txRepo:       txRepo,
		categoryRepo: categoryRepo,
		ruleRepo:     ruleRepo,
		mlClient:     mlClient,
	}
}

func (s *transactionServiceImpl) Create(ctx context.Context, userID string, tx *model.Transaction) (*model.Transaction, error) {
	tx.ID = uuid.New().String()
	tx.UserID = userID
	tx.CreatedAt = time.Now()
	tx.UpdatedAt = time.Now()

	if err := s.Categorize(ctx, userID, tx); err != nil {
		return nil, err
	}

	if err := s.txRepo.Create(ctx, tx); err != nil {
		return nil, err
	}

	return tx, nil
}

func (s *transactionServiceImpl) GetByID(ctx context.Context, userID, id string) (*model.Transaction, error) {
	tx, err := s.txRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Проверка что транзакция принадлежит пользователю
	if tx.UserID != userID {
		return nil, ErrUnauthorized
	}

	return tx, nil
}

func (s *transactionServiceImpl) GetByUserID(ctx context.Context, filter model.TransactionFilter) ([]*model.Transaction, error) {
	// Проверка что пользователь запрашивает свои транзакции
	return s.txRepo.GetByUserID(ctx, filter)
}

func (s *transactionServiceImpl) Update(ctx context.Context, userID string, tx *model.Transaction) (*model.Transaction, error) {
	// Получаем существующую транзакцию
	existing, err := s.txRepo.GetByID(ctx, tx.ID)
	if err != nil {
		return nil, err
	}

	if existing.UserID != userID {
		return nil, ErrUnauthorized
	}

	if err := s.txRepo.Update(ctx, tx); err != nil {
		return nil, err
	}

	return s.txRepo.GetByID(ctx, tx.ID)
}

func (s *transactionServiceImpl) Delete(ctx context.Context, userID, id string) error {
	tx, err := s.txRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if tx.UserID != userID {
		return ErrUnauthorized
	}

	return s.txRepo.Delete(ctx, id)
}

// Categorize выполняет автоматическую категоризацию транзакции
// Алгоритм:
// 1. Проверяем правила пользователя (keyword matching)
// 2. Если правило найдено - используем его категорию
// 3. Если нет - вызываем ML сервис
func (s *transactionServiceImpl) Categorize(ctx context.Context, userID string, tx *model.Transaction) error {
	if tx.Description == "" {
		return nil
	}

	// Получаем правила пользователя
	rules, err := s.ruleRepo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}

	// Ищем совпадение по ключевым словам (rule-based)
	description := strings.ToUpper(tx.Description)
	for _, rule := range rules {
		keyword := strings.ToUpper(rule.Keyword)
		if strings.Contains(description, keyword) {
			tx.CategoryID = &rule.CategoryID
			tx.IsConfirmed = true
			return nil
		}
	}

	// Правило не найдено - вызываем ML сервис
	if s.mlClient != nil {
		result, err := s.mlClient.Categorize(ctx, tx.Description, tx.Amount, tx.Currency)
		if err != nil {
			// ML сервис недоступен - оставляем без категории
			tx.IsConfirmed = false
			return nil
		}

		// Если уверенность высокая (>0.5) - присваиваем категорию
		if result.Confidence > 0.5 {
			categoryID := int(result.CategoryID)
			tx.CategoryID = &categoryID
			tx.IsConfirmed = true
		} else {
			// Низкая уверенность - требуем подтверждения
			tx.IsConfirmed = false
		}
	} else {
		// ML сервис не настроен - оставляем без категории
		tx.IsConfirmed = false
	}

	return nil
}

func (s *transactionServiceImpl) CreateRule(ctx context.Context, userID string, keyword string, categoryID int) (*model.UserCategoryRule, error) {
	// Проверяем что категория существует
	_, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		return nil, err
	}

	rule := &model.UserCategoryRule{
		UserID:     userID,
		Keyword:    keyword,
		CategoryID: categoryID,
	}

	if err := s.ruleRepo.Create(ctx, rule); err != nil {
		return nil, err
	}

	return rule, nil
}

func (s *transactionServiceImpl) GetRules(ctx context.Context, userID string) ([]*model.UserCategoryRule, error) {
	return s.ruleRepo.GetByUserID(ctx, userID)
}

func (s *transactionServiceImpl) DeleteRule(ctx context.Context, userID, ruleID string) error {
	// Получаем правило
	rules, err := s.ruleRepo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}

	// Проверяем что правило принадлежит пользователю
	for _, rule := range rules {
		if rule.ID == ruleID {
			return s.ruleRepo.Delete(ctx, ruleID)
		}
	}

	return ErrUnauthorized
}

func (s *transactionServiceImpl) GetCategories(ctx context.Context) ([]*model.Category, error) {
	return s.categoryRepo.GetAll(ctx)
}
