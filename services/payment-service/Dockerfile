# Stage 1: Builder
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY pkg/go.mod pkg/go.sum* ./pkg/
COPY proto/go.mod proto/go.sum* ./proto/
COPY services/payment-service/go.mod services/payment-service/go.sum* ./services/payment-service/

COPY pkg/ ./pkg/
COPY proto/ ./proto/
COPY services/payment-service/ ./services/payment-service/

WORKDIR /app/services/payment-service

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o payment-service cmd/main.go

# Stage 2: Runtime
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/services/payment-service/payment-service .
COPY --from=builder /app/services/payment-service/migrations ./migrations

EXPOSE 8082

CMD ["./payment-service"]
