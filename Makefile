# ==============================================================
# Makefile — Order Processing System
# ==============================================================

.PHONY: help nats-up nats-down migrate-up migrate-down seed proto-gen \
        tidy-all build-all

# Default target
help:
	@echo ""
	@echo "  Order Processing System — Available Commands"
	@echo "  ─────────────────────────────────────────────"
	@echo "  make nats-up        Start NATS JetStream via Docker"
	@echo "  make nats-down      Stop NATS"
	@echo "  make migrate-up     Run all database migrations (up)"
	@echo "  make migrate-down   Rollback all database migrations"
	@echo "  make seed           Seed inventory products"
	@echo "  make proto-gen      Generate Go code from .proto files"
	@echo "  make tidy-all       Run 'go mod tidy' for all services"
	@echo "  make build-all      Build all service binaries"
	@echo ""

# ── NATS ──────────────────────────────────────────────────────
nats-up:
	docker compose up -d nats
	@echo "✅ NATS JetStream started. Monitoring: http://localhost:8222"

nats-down:
	docker compose down

# ── Migrations ────────────────────────────────────────────────
migrate-up:
	@bash scripts/migrate-all.sh up

migrate-down:
	@bash scripts/migrate-all.sh down

# ── Seed ──────────────────────────────────────────────────────
seed:
	@source .env && psql "$$INVENTORY_DB_URL" -f services/inventory-service/migrations/seed.sql
	@echo "✅ Seed data loaded for inventory-service"

# ── Proto Generation ──────────────────────────────────────────
# Requires: protoc + protoc-gen-go + protoc-gen-go-grpc in PATH
proto-gen:
	@echo "Generating Go code from proto files..."
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		proto/inventory/inventory.proto
	@echo "✅ Proto generation complete"

# ── Go Module Tidy ────────────────────────────────────────────
tidy-all:
	@for svc in order-service inventory-service payment-service notification-service; do \
		echo "→ go mod tidy: $$svc"; \
		cd services/$$svc && go mod tidy && cd ../..; \
	done
	@echo "✅ go mod tidy complete for all services"

# ── Build ─────────────────────────────────────────────────────
build-all:
	@mkdir -p bin
	@for svc in order-service inventory-service payment-service notification-service; do \
		echo "→ building $$svc..."; \
		cd services/$$svc && go build -o ../../bin/$$svc ./cmd/... && cd ../..; \
	done
	@echo "✅ All binaries built in ./bin/"
