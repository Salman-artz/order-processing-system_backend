# Order Processing System - Backend

Backend event-driven untuk pemrosesan order dengan pola **Saga choreography**. Sistem terdiri dari beberapa microservice Go yang berkomunikasi melalui REST, gRPC, dan NATS JetStream.

## Status Project

Versi saat ini sudah memiliki fondasi dan alur inti berikut:

- Order Service: register/login JWT, membuat order, melihat order milik customer, dan admin order timeline.
- Inventory Service: daftar produk, reservasi stok, pelepasan stok, REST, dan gRPC.
- Payment Service: simulasi proses pembayaran berbasis event.
- Notification Service: pencatatan event dan update order melalui WebSocket.
- PostgreSQL terpisah untuk setiap service.
- Auto-migration saat service mulai.
- Auto-seed produk inventory saat database masih kosong.
- Docker Compose untuk menjalankan backend beserta PostgreSQL dan NATS.

## Arsitektur

```text
                         +-------------------+
 Customer / Admin ------>|  Order Service    | :8080
        REST              +---------+---------+
                                   |
                              NATS JetStream
               +-----------------+-----------------+
               |                 |                 |
       +-------v-------+ +-------v-------+ +-------v--------+
       | Inventory     | | Payment       | | Notification   |
       | :8081 / gRPC  | | :8082         | | :8083 / WS     |
       +-------+-------+ +-------+-------+ +--------+-------+
               |                 |                  |
          inventory_db       payment_db       notification_db

                         order_db
```

### Alur Saga

1. Customer membuat order melalui `POST /api/v1/orders`.
2. Order Service menyimpan order sebagai `PENDING` dan menerbitkan `order.created`.
3. Inventory Service memeriksa serta mereservasi stok.
4. Jika stok tersedia, Order Service menerbitkan `payment.requested`.
5. Payment Service menjalankan pembayaran mock dan menerbitkan hasil pembayaran.
6. Order Service mengubah status menjadi `COMPLETED` atau `CANCELLED`.
7. Notification Service mencatat event dan mengirim update melalui WebSocket.

## Tech Stack

| Area | Teknologi |
|---|---|
| Bahasa | Go 1.23+ |
| HTTP | Fiber v2 |
| RPC | gRPC + Protocol Buffers |
| Event broker | NATS JetStream |
| Database | PostgreSQL 14+ |
| Migration | golang-migrate |
| Authentication | JWT |
| Logging | zerolog |
| Metrics | Prometheus client |
| Deployment lokal | Docker Compose |

## Struktur Repository

```text
services/
  order-service/         # API order, auth, dan saga consumer
  inventory-service/     # Produk, stok, REST, dan gRPC
  payment-service/       # Pembayaran mock berbasis event
  notification-service/  # Log notifikasi dan WebSocket
pkg/
  events/                # Subject dan payload event
  jwtutil/               # Helper JWT
  logger/                # Shared logger
  natsclient/            # Shared NATS client
  postgres/              # Connector dan migration helper
proto/                   # Definisi gRPC inventory
scripts/                 # Database creation dan migration script
docker-compose.yml       # Full local stack
Makefile                 # Perintah development
```

## Prasyarat

- Go 1.23 atau lebih baru
- Docker Desktop dan Docker Compose v2
- PostgreSQL 14+ jika menjalankan dependency secara native
- `golang-migrate` jika menjalankan migration secara manual
- `protoc` dan plugin Go jika ingin generate ulang kode protobuf

## Menjalankan Full Stack dengan Docker

Cara ini menjalankan NATS, empat database PostgreSQL, dan seluruh service Go.

```bash
cp .env.docker.example .env.docker
docker compose --env-file .env.docker up -d --build
docker compose ps
```

Endpoint utama:

| Service | URL / Port |
|---|---|
| Order REST API | `http://localhost:8080` |
| Inventory REST API | `http://localhost:8081` |
| Payment Service | `http://localhost:8082` |
| Notification WebSocket | `ws://localhost:8083/ws/orders/:orderId` |
| NATS client | `nats://localhost:4222` |
| NATS monitoring | `http://localhost:8222` |

Semua service menjalankan migration saat startup. Inventory Service otomatis mengisi data produk jika tabel `products` masih kosong.

Melihat log:

```bash
docker compose logs -f order-service
docker compose logs -f inventory-service
```

Menghentikan stack:

```bash
docker compose down
```

Untuk menghapus container sekaligus seluruh data database Docker:

```bash
docker compose down -v
```

## Menjalankan Secara Native

1. Salin konfigurasi:

   ```bash
   cp .env.example .env
   ```

2. Isi koneksi database dan secret JWT di `.env`.
3. Buat database PostgreSQL:

   ```bash
   psql -U postgres -h localhost -f scripts/create-databases.sql
   ```

4. Jalankan NATS:

   ```bash
   docker compose up -d nats
   ```

5. Jalankan migration manual bila diperlukan:

   ```bash
   bash scripts/migrate-all.sh up
   ```

6. Jalankan service dari folder masing-masing:

   ```bash
   cd services/order-service
   go run ./cmd
   ```

Untuk development, gunakan beberapa terminal untuk service inventory, payment, dan notification.

## Environment Variables

Template tersedia di `.env.example` dan `.env.docker.example`. Variable utama:

| Variable | Fungsi |
|---|---|
| `ORDER_DB_URL` | Koneksi database order |
| `INVENTORY_DB_URL` | Koneksi database inventory |
| `PAYMENT_DB_URL` | Koneksi database payment |
| `NOTIFICATION_DB_URL` | Koneksi database notification |
| `NATS_URL` | URL NATS JetStream |
| `JWT_SECRET` | Secret signing JWT |
| `ORDER_SERVICE_PORT` | Port Order Service, default `8080` |
| `INVENTORY_HTTP_PORT` | Port REST Inventory, default `8081` |
| `PAYMENT_SERVICE_PORT` | Port Payment Service, default `8082` |
| `NOTIFICATION_SERVICE_PORT` | Port Notification, default `8083` |
| `INVENTORY_GRPC_PORT` | Port gRPC Inventory |

Jangan commit `.env` atau secret production.

## Endpoint Utama

### Auth

- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`

### Customer

- `POST /api/v1/orders`
- `GET /api/v1/orders`
- `GET /api/v1/orders/:id`

### Admin

- `GET /api/v1/admin/orders`
- `GET /api/v1/admin/orders/:id/timeline`

### Inventory

- `GET /api/v1/products`
- `GET /health` tersedia pada setiap HTTP service.

Gunakan header berikut untuk endpoint yang membutuhkan autentikasi:

```text
Authorization: Bearer <JWT_TOKEN>
```

## Event Subjects

| Subject | Publisher | Consumer utama |
|---|---|---|
| `order.created` | Order | Inventory |
| `stock.reserved` | Inventory | Order |
| `stock.rejected` | Inventory | Order |
| `payment.requested` | Order | Payment |
| `payment.success` | Payment | Order, Notification |
| `payment.failed` | Payment | Order, Notification |
| `order.cancelled` | Order | Inventory, Notification |
| `order.completed` | Order | Notification |

## Make Commands

```bash
make help
make nats-up
make nats-down
make migrate-up
make migrate-down
make seed
make proto-gen
make tidy-all
make build-all
```

## Catatan Pengembangan

- Payment saat ini masih berupa mock service.
- Saga berjalan melalui event NATS; pastikan NATS aktif sebelum service dimulai.
- Database dipisahkan per service untuk menjaga ownership data.
- `docker compose down -v` bersifat destruktif terhadap data lokal Docker.
