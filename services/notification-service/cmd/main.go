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
	"github.com/gofiber/websocket/v2"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/order-processing-system/pkg/events"
	"github.com/order-processing-system/pkg/logger"
	"github.com/order-processing-system/pkg/natsclient"
	"github.com/order-processing-system/pkg/postgres"
	"github.com/order-processing-system/notification-service/internal/event"
	"github.com/order-processing-system/notification-service/internal/handler"
	"github.com/order-processing-system/notification-service/internal/repository"
	"github.com/order-processing-system/notification-service/internal/service"
)

func main() {
	if err := godotenv.Load("../../.env"); err != nil {
		fmt.Println("⚠️  .env not found, using system environment variables")
	}

	logCfg := logger.Config{
		ServiceName: "notification-service",
		Level:       os.Getenv("LOG_LEVEL"),
		Pretty:      os.Getenv("APP_ENV") != "production",
	}
	log := logger.Init(logCfg)

	dbURL := os.Getenv("NOTIFICATION_DB_URL")
	
	// Auto-migration
	if err := postgres.RunMigration(dbURL, "migrations"); err != nil {
		log.Fatal().Err(err).Msg("Failed to run migrations")
	}

	pool := postgres.MustConnect(context.Background(), postgres.Config{DSN: dbURL})
	defer pool.Close()
	log.Info().Msg("Connected to PostgreSQL")

	repo := repository.NewRepository(pool)
	notifSvc := service.NewNotificationService(repo)

	natsURL := os.Getenv("NATS_URL")
	natsClient := natsclient.MustConnect(natsclient.Config{
		URL:  natsURL,
		Name: "notification-service",
	})
	defer natsClient.Close()
	log.Info().Msg("Connected to NATS")

	// Ensure stream exists
	_, err := natsClient.EnsureStream(context.Background(), jetstream.StreamConfig{
		Name:     events.StreamName,
		Subjects: []string{"order.*", "stock.*", "payment.*"},
		Storage:  jetstream.FileStorage,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to ensure NATS stream")
	}

	consumer := event.NewConsumer(natsClient.JS, notifSvc)
	if err := consumer.Start(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("Failed to start NATS consumer")
	}

	app := fiber.New(fiber.Config{
		AppName:      "Notification Service v1.0",
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

	wsHandler := handler.NewNotificationWSHandler(notifSvc)
	restHandler := handler.NewNotificationRESTHandler(notifSvc)

	// REST endpoints
	app.Get("/api/v1/admin/orders/:id/timeline", restHandler.GetLogs)

	// WebSocket endpoint
	app.Use("/ws", handler.UpgradeMiddleware)
	app.Get("/ws/orders/:id", websocket.New(wsHandler.HandleWS))

	httpPort := os.Getenv("NOTIFICATION_HTTP_PORT")
	if httpPort == "" {
		httpPort = "8083"
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Info().Msgf("🚀 Notification Service REST & WS listening on :%s", httpPort)
		if err := app.Listen(fmt.Sprintf(":%s", httpPort)); err != nil {
			log.Fatal().Err(err).Msg("Server error")
		}
	}()

	<-quit
	log.Info().Msg("⏳ Shutting down Notification Service...")
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = app.ShutdownWithContext(ctx)
	
	log.Info().Msg("✅ Notification Service stopped.")
}
