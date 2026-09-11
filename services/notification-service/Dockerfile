# Stage 1: Builder
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY pkg/go.mod pkg/go.sum* ./pkg/
COPY proto/go.mod proto/go.sum* ./proto/
COPY services/notification-service/go.mod services/notification-service/go.sum* ./services/notification-service/

COPY pkg/ ./pkg/
COPY proto/ ./proto/
COPY services/notification-service/ ./services/notification-service/

WORKDIR /app/services/notification-service

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o notification-service cmd/main.go

# Stage 2: Runtime
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/services/notification-service/notification-service .
COPY --from=builder /app/services/notification-service/migrations ./migrations

EXPOSE 8083

CMD ["./notification-service"]
