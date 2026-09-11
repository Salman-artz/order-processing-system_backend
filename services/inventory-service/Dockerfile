# Stage 1: Builder
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY pkg/go.mod pkg/go.sum* ./pkg/
COPY proto/go.mod proto/go.sum* ./proto/
COPY services/inventory-service/go.mod services/inventory-service/go.sum* ./services/inventory-service/

COPY pkg/ ./pkg/
COPY proto/ ./proto/
COPY services/inventory-service/ ./services/inventory-service/

WORKDIR /app/services/inventory-service

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o inventory-service cmd/main.go

# Stage 2: Runtime
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/services/inventory-service/inventory-service .
COPY --from=builder /app/services/inventory-service/migrations ./migrations

EXPOSE 8081
EXPOSE 50051

CMD ["./inventory-service"]
