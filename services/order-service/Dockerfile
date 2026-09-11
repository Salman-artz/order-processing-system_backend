# Stage 1: Builder
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go.work if you use workspaces, else copy mod files
COPY pkg/go.mod pkg/go.sum* ./pkg/
COPY proto/go.mod proto/go.sum* ./proto/
COPY services/order-service/go.mod services/order-service/go.sum* ./services/order-service/

# Copy source code
COPY pkg/ ./pkg/
COPY proto/ ./proto/
COPY services/order-service/ ./services/order-service/

WORKDIR /app/services/order-service

# Download dependencies
RUN go mod download

# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o order-service cmd/main.go

# Stage 2: Runtime
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/services/order-service/order-service .
# Copy migrations for auto-migrate on startup
COPY --from=builder /app/services/order-service/migrations ./migrations

EXPOSE 8080

CMD ["./order-service"]
