package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/grpc"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	"github.com/order-processing-system/inventory-service/internal/event"
	"github.com/order-processing-system/inventory-service/internal/handler"
	"github.com/order-processing-system/inventory-service/internal/repository"
	"github.com/order-processing-system/inventory-service/internal/service"
	"github.com/order-processing-system/pkg/events"
	"github.com/order-processing-system/pkg/logger"
	"github.com/order-processing-system/pkg/natsclient"
	"github.com/order-processing-system/pkg/postgres"
	inventory "github.com/order-processing-system/proto/inventory"
)

func main() {
	if err := godotenv.Load("../../.env"); err != nil {
		fmt.Println("⚠️  .env not found, using system environment variables")
	}

	logCfg := logger.Config{
		ServiceName: "inventory-service",
		Level:       os.Getenv("LOG_LEVEL"),
		Pretty:      os.Getenv("APP_ENV") != "production",
	}
	log := logger.Init(logCfg)

	dbURL := os.Getenv("INVENTORY_DB_URL")
	
	// Auto-migration
	if err := postgres.RunMigration(dbURL, "migrations"); err != nil {
		fmt.Printf("ERROR: %v\n", err); log.Fatal().Err(err).Msg("Failed to run migrations")
	}

	pool := postgres.MustConnect(context.Background(), postgres.Config{DSN: dbURL})
	defer pool.Close()
	log.Info().Msg("Connected to PostgreSQL")

	// Auto-seed
	var count int
	if err := pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM products").Scan(&count); err == nil && count == 0 {
		log.Info().Msg("Database is empty. Running auto-seed...")
		seedBytes, err := os.ReadFile("migrations/seed.sql")
		if err == nil {
			_, err = pool.Exec(context.Background(), string(seedBytes))
			if err != nil {
				log.Error().Err(err).Msg("Failed to execute seed.sql")
			} else {
				log.Info().Msg("Seed data successfully inserted.")
			}
		} else {
			log.Error().Err(err).Msg("Failed to read seed.sql")
		}
	}

	repo := repository.NewRepository(pool)

	natsURL := os.Getenv("NATS_URL")
	natsClient := natsclient.MustConnect(natsclient.Config{
		URL:  natsURL,
		Name: "inventory-service",
	})
	defer natsClient.Close()
	log.Info().Msg("Connected to NATS")

	_, err := natsClient.EnsureStream(context.Background(), jetstream.StreamConfig{
		Name:     events.StreamName,
		Subjects: []string{"order.*", "stock.*", "payment.*"},
		Storage:  jetstream.FileStorage,
	})
	if err != nil {
		fmt.Printf("ERROR: %v\n", err); log.Fatal().Err(err).Msg("Failed to ensure NATS stream")
	}

	publisher := event.NewPublisher(natsClient.JS)
	invSvc := service.NewInventoryService(repo, publisher)
	consumer := event.NewConsumer(natsClient.JS, invSvc)

	if err := consumer.Start(context.Background()); err != nil {
		fmt.Printf("ERROR: %v\n", err); log.Fatal().Err(err).Msg("Failed to start NATS consumer")
	}

	// gRPC Server setup
	grpcPort := os.Getenv("INVENTORY_GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		fmt.Printf("ERROR: %v\n", err); log.Fatal().Err(err).Msg("Failed to listen for gRPC")
	}

	grpcServer := grpc.NewServer()
	inventoryHandler := handler.NewInventoryGRPCServer(invSvc, repo)
	inventory.RegisterInventoryServiceServer(grpcServer, inventoryHandler)

	// Setup Fiber App for REST endpoint
	app := fiber.New(fiber.Config{
		AppName:      "Inventory Service v1.0",
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

	restHandler := handler.NewInventoryRESTHandler(repo)
	v1 := app.Group("/api/v1")
	v1.Get("/products", restHandler.ListProducts)
	v1.Post("/products", restHandler.CreateProduct)
	v1.Put("/products/:id", restHandler.UpdateProduct)
	v1.Patch("/products/:id/stock", restHandler.RestockProduct)
	v1.Post("/products/:id/restock", restHandler.RestockProduct)
	v1.Post("/products/:id/stock", restHandler.RestockProduct)
	v1.Delete("/products/:id", restHandler.DeleteProduct)

	httpPort := os.Getenv("INVENTORY_HTTP_PORT")
	if httpPort == "" {
		httpPort = "8081" // Different from order-service (8080)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start Fiber
	go func() {
		log.Info().Msgf("🚀 Inventory Service REST listening on :%s", httpPort)
		if err := app.Listen(fmt.Sprintf(":%s", httpPort)); err != nil {
			fmt.Printf("ERROR: %v\n", err); log.Fatal().Err(err).Msg("REST server error")
		}
	}()

	// Start gRPC
	go func() {
		log.Info().Msgf("🚀 Inventory Service gRPC listening on :%s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			fmt.Printf("ERROR: %v\n", err); log.Fatal().Err(err).Msg("gRPC server error")
		}
	}()

	<-quit
	log.Info().Msg("⏳ Shutting down Inventory Service...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = app.ShutdownWithContext(ctx)

	grpcServer.GracefulStop()
	log.Info().Msg("✅ Inventory Service stopped.")
}
