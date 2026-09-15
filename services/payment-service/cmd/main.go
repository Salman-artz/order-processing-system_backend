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
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/order-processing-system/pkg/events"
	"github.com/order-processing-system/pkg/logger"
	"github.com/order-processing-system/pkg/natsclient"
	"github.com/order-processing-system/pkg/postgres"
	"github.com/order-processing-system/payment-service/internal/event"
	"github.com/order-processing-system/payment-service/internal/handler"
	"github.com/order-processing-system/payment-service/internal/repository"
	"github.com/order-processing-system/payment-service/internal/service"
)

func main() {
	if err := godotenv.Load("../../.env"); err != nil {
		fmt.Println("⚠️  .env not found, using system environment variables")
	}

	logCfg := logger.Config{
		ServiceName: "payment-service",
		Level:       os.Getenv("LOG_LEVEL"),
		Pretty:      os.Getenv("APP_ENV") != "production",
	}
	log := logger.Init(logCfg)

	dbURL := os.Getenv("PAYMENT_DB_URL")
	
	// Auto-migration
	if err := postgres.RunMigration(dbURL, "migrations"); err != nil {
		log.Fatal().Err(err).Msg("Failed to run migrations")
	}
	
	pool := postgres.MustConnect(context.Background(), postgres.Config{DSN: dbURL})
	defer pool.Close()
	log.Info().Msg("Connected to PostgreSQL")

	repo := repository.NewRepository(pool)

	natsURL := os.Getenv("NATS_URL")
	natsClient := natsclient.MustConnect(natsclient.Config{
		URL:  natsURL,
		Name: "payment-service",
	})
	defer natsClient.Close()
	log.Info().Msg("Connected to NATS")

	_, err := natsClient.EnsureStream(context.Background(), jetstream.StreamConfig{
		Name:     events.StreamName,
		Subjects: []string{"order.*", "stock.*", "payment.*"},
		Storage:  jetstream.FileStorage,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to ensure NATS stream")
	}

	publisher := event.NewPublisher(natsClient.JS)
	paySvc := service.NewPaymentService(repo, publisher)
	consumer := event.NewConsumer(natsClient.JS, paySvc)

	if err := consumer.Start(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("Failed to start NATS consumer")
	}

	// Setup Fiber App for REST endpoint (Admin/Debugging)
	app := fiber.New(fiber.Config{
		AppName:      "Payment Service v1.0",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS, HEAD",
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	restHandler := handler.NewPaymentRESTHandler(paySvc)

	v1 := app.Group("/api/v1")
	admin := v1.Group("/admin")
	admin.Get("/payments/:order_id", restHandler.GetPaymentsByOrder)

	// Midtrans Gateway Routes
	payments := v1.Group("/payments")
	payments.Post("/midtrans/snap", restHandler.CreateSnap)
	payments.Post("/midtrans/notification", restHandler.MidtransNotification)
	payments.Post("/midtrans/simulate", restHandler.SimulatePayment)

	httpPort := os.Getenv("PAYMENT_HTTP_PORT")
	if httpPort == "" {
		httpPort = "8082" // Different from order(8080) and inventory(8081)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start Fiber
	go func() {
		log.Info().Msgf("🚀 Payment Service REST listening on :%s", httpPort)
		if err := app.Listen(fmt.Sprintf(":%s", httpPort)); err != nil {
			log.Fatal().Err(err).Msg("REST server error")
		}
	}()

	<-quit
	log.Info().Msg("⏳ Shutting down Payment Service...")
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = app.ShutdownWithContext(ctx)
	
	log.Info().Msg("✅ Payment Service stopped.")
}
