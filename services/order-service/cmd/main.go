package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go/jetstream"
	
	"github.com/order-processing-system/pkg/events"
	"github.com/order-processing-system/pkg/logger"
	"github.com/order-processing-system/pkg/natsclient"
	"github.com/order-processing-system/pkg/postgres"
	"github.com/order-processing-system/order-service/internal/event"
	"github.com/order-processing-system/order-service/internal/handler"
	"github.com/order-processing-system/order-service/internal/repository"
	"github.com/order-processing-system/order-service/internal/service"
)

func main() {
	if err := godotenv.Load("../../.env"); err != nil {
		fmt.Println("⚠️  .env not found, using system environment variables")
	}

	// Init Logger
	logCfg := logger.Config{
		ServiceName: "order-service",
		Level:       os.Getenv("LOG_LEVEL"),
		Pretty:      os.Getenv("APP_ENV") != "production",
	}
	log := logger.Init(logCfg)

	// Connect to Database
	dbURL := os.Getenv("ORDER_DB_URL")
	
	// Auto-migration
	if err := postgres.RunMigration(dbURL, "migrations"); err != nil {
		fmt.Printf("ERROR: %v\n", err); log.Fatal().Err(err).Msg("Failed to run migrations")
	}
	
	pool := postgres.MustConnect(context.Background(), postgres.Config{DSN: dbURL})
	defer pool.Close()
	log.Info().Msg("Connected to PostgreSQL")

	// Init Repository
	repo := repository.NewRepository(pool)

	// Connect to NATS JetStream
	natsURL := os.Getenv("NATS_URL")
	natsClient := natsclient.MustConnect(natsclient.Config{
		URL:  natsURL,
		Name: "order-service",
	})
	defer natsClient.Close()
	log.Info().Msg("Connected to NATS")

	// Ensure Stream exists
	_, err := natsClient.EnsureStream(context.Background(), jetstream.StreamConfig{
		Name:     events.StreamName,
		Subjects: []string{"order.*", "stock.*", "payment.*"},
		Storage:  jetstream.FileStorage,
	})
	if err != nil {
		fmt.Printf("ERROR: %v\n", err); log.Fatal().Err(err).Msg("Failed to ensure NATS stream")
	}

	// Init Publisher and Consumer
	publisher := event.NewPublisher(natsClient.JS)
	consumer := event.NewConsumer(natsClient.JS, repo, publisher)

	if err := consumer.Start(context.Background()); err != nil {
		fmt.Printf("ERROR: %v\n", err); log.Fatal().Err(err).Msg("Failed to start NATS consumer")
	}

	// Init Services
	authSvc := service.NewAuthService(repo)
	orderSvc := service.NewOrderService(repo, publisher)

	// Init Handlers
	authHandler := handler.NewAuthHandler(authSvc)
	orderHandler := handler.NewOrderHandler(orderSvc)

	// Setup Fiber App
	app := fiber.New(fiber.Config{
		AppName:      "Order Service v1.0",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD",
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Setup Routes
	v1 := app.Group("/api/v1")
	
	// Products proxy to inventory-service
	invHTTPURL := os.Getenv("INVENTORY_HTTP_URL")
	if invHTTPURL == "" {
		invHTTPURL = "http://inventory-service:8081"
	}
	v1.All("/products*", func(c *fiber.Ctx) error {
		return proxy.Do(c, fmt.Sprintf("%s%s", invHTTPURL, c.OriginalURL()))
	})

	// Payments proxy to payment-service
	payHTTPURL := os.Getenv("PAYMENT_HTTP_URL")
	if payHTTPURL == "" {
		payHTTPURL = "http://payment-service:8082"
	}
	v1.All("/payments/*", func(c *fiber.Ctx) error {
		return proxy.Do(c, fmt.Sprintf("%s%s", payHTTPURL, c.OriginalURL()))
	})

	// Auth routes
	auth := v1.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)

	// Order routes (Customer)
	orders := v1.Group("/orders", handler.JWTMiddleware)
	orders.Post("/", orderHandler.CreateOrder)
	orders.Get("/", orderHandler.ListCustomerOrders)
	orders.Get("/:id", orderHandler.GetOrder)

	// Admin routes
	admin := v1.Group("/admin", handler.JWTMiddleware, handler.RequireRole("admin"))
	admin.Get("/orders", orderHandler.ListAllOrders)
	admin.Get("/orders/:id/timeline", orderHandler.GetOrderTimeline)

	port := os.Getenv("ORDER_SERVICE_PORT")
	if port == "" {
		port = "8080"
	}

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Info().Msgf("🚀 Order Service listening on :%s", port)
		if err := app.Listen(fmt.Sprintf(":%s", port)); err != nil {
			fmt.Printf("ERROR: %v\n", err); log.Fatal().Err(err).Msg("Server error")
		}
	}()

	<-quit
	log.Info().Msg("⏳ Shutting down Order Service...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Warn().Err(err).Msg("Shutdown error")
	}
	log.Info().Msg("✅ Order Service stopped.")
}
