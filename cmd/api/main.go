package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/hibiken/asynq"
	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/gibbon/finace-dashboard/docs"
	"github.com/gibbon/finace-dashboard/internal/config"
	"github.com/gibbon/finace-dashboard/internal/grpc_client"
	"github.com/gibbon/finace-dashboard/internal/handlers"
	appMiddleware "github.com/gibbon/finace-dashboard/internal/middleware"
	"github.com/gibbon/finace-dashboard/internal/repository"
	"github.com/gibbon/finace-dashboard/internal/service"
	"github.com/gibbon/finace-dashboard/pkg/jwt"
	"github.com/jackc/pgx/v5/pgxpool"
)

// @title Personal Finance Dashboard API
// @version 1.0
// @description Backend API для персонального финансового дашборда
// @host localhost:8080
// @BasePath /api/v1
// @schemes http https

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dbPool, err := pgxpool.New(context.Background(), cfg.Database.DSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := dbPool.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	userRepo := repository.NewPostgresUserRepository(dbPool)
	txRepo := repository.NewPostgresTransactionRepository(dbPool)
	categoryRepo := repository.NewPostgresCategoryRepository(dbPool)
	ruleRepo := repository.NewPostgresUserCategoryRuleRepository(dbPool)

	authService := service.NewAuthService(userRepo, service.AuthServiceConfig{
		JWTSecret:     cfg.JWT.Secret,
		AccessExpiry:  cfg.JWT.AccessExpiry,
		RefreshExpiry: cfg.JWT.RefreshExpiry,
	})

	// Создаём ML gRPC клиент (опционально)
	var mlClient *grpc_client.MLClient
	mlClient, err = grpc_client.NewMLClient(grpc_client.MLClientConfig{
		Host: cfg.MLService.Host,
		Port: cfg.MLService.Port,
	})
	if err != nil {
		log.Printf("Warning: failed to connect to ML service: %v", err)
		log.Println("ML categorization will be disabled, rule-based categorization will be used")
	}

	txService := service.NewTransactionService(txRepo, categoryRepo, ruleRepo, mlClient)

	// Создаём JWT менеджер
	jwtManager := jwt.NewManager(cfg.JWT.Secret, cfg.JWT.AccessExpiry, cfg.JWT.RefreshExpiry)

	// Создаём asynq клиент для очереди задач
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr: cfg.Redis.Address(),
		Password: cfg.Redis.Password,
		DB:       0,
	})
	defer asynqClient.Close()

	// Создаём обработчики
	importHandler := handlers.NewImportHandler(asynqClient)
	authHandler := handlers.NewAuthHandler(authService)
	authMiddleware := appMiddleware.NewAuthMiddleware(jwtManager)
	txHandler := handlers.NewTransactionHandler(txService)
	categoryHandler := handlers.NewCategoryHandler(txService)
	categoryRuleHandler := handlers.NewCategoryRuleHandler(txService)

	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Swagger UI
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("#swagger-ui"),
	))

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/logout", authHandler.Logout)
		})

		// Категории
		// TODO: сделать protected
		r.Get("/categories", categoryHandler.GetAll)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Middleware)

			// User info
			r.Get("/me", func(w http.ResponseWriter, r *http.Request) {
				userID, _ := appMiddleware.GetUserIDFromContext(r.Context())
				email, _ := appMiddleware.GetEmailFromContext(r.Context())
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"user_id": "%s", "email": "%s"}`, userID, email)
			})

			// Транзакции
			r.Route("/transactions", func(r chi.Router) {
				r.Post("/", txHandler.Create)
				r.Get("/", txHandler.GetAll)
				r.Get("/{id}", txHandler.GetByID)
				r.Put("/{id}", txHandler.Update)
				r.Delete("/{id}", txHandler.Delete)
			})

			// Правила категорий
			r.Route("/category-rules", func(r chi.Router) {
				r.Post("/", categoryRuleHandler.Create)
				r.Get("/", categoryRuleHandler.GetAll)
				r.Delete("/{id}", categoryRuleHandler.Delete)
			})

			// Импорт транзакций
			r.Route("/imports", func(r chi.Router) {
				r.Post("/", importHandler.Import)
				r.Post("/sync", importHandler.ImportSync)
				r.Get("/status", importHandler.GetImportStatus)
			})
		})
	})

	server := &http.Server{
		Addr:         cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		log.Println("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Закрываем ML клиент
		if mlClient != nil {
			mlClient.Close()
		}

		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("Server starting on port %s", cfg.Server.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}
