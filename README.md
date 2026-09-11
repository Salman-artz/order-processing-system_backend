# Order Processing System — Backend

Event-driven microservices backend built with Go, NATS JetStream, and PostgreSQL.

> **Fase saat ini: Fase 1 (Scaffolding + Database)** — Business logic belum diimplementasi.

## Arsitektur

```
Customer / Admin (Next.js — project terpisah)
           │ REST/HTTP
           ▼
   ┌───────────────────┐
   │   Order Service   │ :8080  ← entry point utama
   │   (PostgreSQL:    │
   │    order_db)      │
   └──────┬────────────┘
          │ gRPC (sync check)
          ▼
   ┌───────────────────┐        ┌─────────────────────┐
   │ Inventory Service │ :8081  │  Payment Service     │ :8082
   │ (inventory_db)    │        │  (payment_db, mock)  │
   └───────────────────┘        └─────────────────────┘
                                          │
   ┌───────────────────┐                  │
   │Notification Svc   │ :8083 ←──────────┘
   │(notification_db)  │
   └───────────────────┘

   Message Broker: NATS JetStream (Docker, port 4222)
   Database: PostgreSQL lokal localhost:5432 (4 database terpisah)
```

### Alur Saga (Choreography)
1. `POST /orders` → **Order Service** simpan `PENDING` → publish `order.created`
2. **Inventory Service** consume → reserve stok → publish `stock.reserved` / `stock.rejected`
3. **Order Service** consume `stock.reserved` → update `AWAITING_PAYMENT` → publish `payment.requested`
4. **Payment Service** consume → mock payment → publish `payment.success` / `payment.failed`
5. **Order Service** consume hasil:
   - Sukses → `COMPLETED` → publish `order.completed`
   - Gagal → `CANCELLED` → publish `order.cancelled` (Inventory Service release stok)
6. **Notification Service** consume semua events → log / WebSocket push

---

## Struktur Folder

```
order-processing-system/
├── services/
│   ├── order-service/
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   │   ├── handler/      # HTTP handlers (Fase 2)
│   │   │   ├── repository/   # DB queries (Fase 2)
│   │   │   ├── service/      # Business logic (Fase 2)
│   │   │   ├── event/        # NATS publisher & consumer (Fase 3)
│   │   │   └── saga/         # Saga orchestration (Fase 3-4)
│   │   ├── migrations/
│   │   └── go.mod
│   ├── inventory-service/    (struktur sama)
│   ├── payment-service/      (struktur sama)
│   └── notification-service/ (struktur sama)
├── proto/
│   └── inventory/inventory.proto   # gRPC definitions
├── pkg/
│   ├── logger/      # Shared zerolog logger
│   ├── postgres/    # Shared pgxpool connector
│   ├── natsclient/  # Shared NATS JetStream client
│   ├── jwtutil/     # Shared JWT helper
│   └── events/      # NATS subject constants & payload types
├── scripts/
│   ├── create-databases.sql  # Buat 4 database di Postgres lokal
│   └── migrate-all.sh        # Jalankan semua migrations
├── docker-compose.yml  # NATS JetStream only
├── Makefile
├── .env.example
└── .gitignore
```

---

## Setup Fase 1 — Instruksi Validasi Manual

### Prasyarat
- PostgreSQL 14+ sudah berjalan di `localhost:5432`
- Docker Desktop sudah berjalan
- [`golang-migrate`](https://github.com/golang-migrate/migrate/releases) CLI terinstall
- Go 1.23+

---

### Langkah 1 — Salin dan Isi File `.env`

```bash
# Dari folder order-processing-system/
cp .env.example .env
```

Buka `.env`, ganti `<user>` dan `<password>` dengan kredensial PostgreSQL lokal kamu:
```env
ORDER_DB_URL=postgres://postgres:mypassword@localhost:5432/order_db?sslmode=disable
INVENTORY_DB_URL=postgres://postgres:mypassword@localhost:5432/inventory_db?sslmode=disable
PAYMENT_DB_URL=postgres://postgres:mypassword@localhost:5432/payment_db?sslmode=disable
NOTIFICATION_DB_URL=postgres://postgres:mypassword@localhost:5432/notification_db?sslmode=disable
NATS_URL=nats://localhost:4222
JWT_SECRET=isi-dengan-random-string-minimal-32-karakter
```

---

### Langkah 2 — Buat 4 Database di PostgreSQL Lokal

**Opsi A — via psql CLI:**
```bash
psql -U postgres -h localhost -f scripts/create-databases.sql
```

**Opsi B — via pgAdmin atau TablePlus:**
1. Buka pgAdmin / TablePlus
2. Connect ke `localhost:5432` sebagai superuser (biasanya `postgres`)
3. Buka Query Editor / SQL Editor
4. Paste isi file `scripts/create-databases.sql`
5. Jalankan (Execute / F5)

**Verifikasi database berhasil dibuat:**
```sql
-- Jalankan di psql atau pgAdmin, sambung ke database postgres (bukan yang baru)
SELECT datname FROM pg_database
WHERE datname IN ('order_db', 'inventory_db', 'payment_db', 'notification_db')
ORDER BY datname;
```
Harus muncul 4 baris.

---

### Langkah 3 — Jalankan NATS via Docker

```bash
docker compose up -d
```

**Verifikasi NATS berjalan:**
```bash
# Cek container
docker ps | grep order-nats

# Atau buka browser:
# http://localhost:8222   → NATS monitoring dashboard
```

---

### Langkah 4 — Install golang-migrate (jika belum)

```bash
# Opsi 1: via Go install
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Opsi 2: Download binary dari https://github.com/golang-migrate/migrate/releases
# Letakkan di folder yang ada di PATH (misal C:\Users\<user>\go\bin\)

# Verifikasi:
migrate -version
```

---

### Langkah 5 — Jalankan Semua Migrations

```bash
# Dari folder order-processing-system/
bash scripts/migrate-all.sh
```

> **Windows (Git Bash / WSL):** Jalankan di Git Bash atau WSL.
> **Windows (PowerShell):** Jalankan dengan `bash scripts/migrate-all.sh` di Git Bash, atau install WSL.

Kalau berhasil, output akan seperti:
```
✅ order-service migration 'up' berhasil!
✅ inventory-service migration 'up' berhasil!
✅ payment-service migration 'up' berhasil!
✅ notification-service migration 'up' berhasil!
```

---

### Langkah 6 — Verifikasi Tabel Berhasil Dibuat

Sambungkan ke setiap database dan jalankan `\dt`:

**Via psql:**
```bash
# order_db
psql -U postgres -h localhost -d order_db -c "\dt"
# Harus muncul: orders, order_items, saga_logs

# inventory_db
psql -U postgres -h localhost -d inventory_db -c "\dt"
# Harus muncul: products, stock_reservations

# payment_db
psql -U postgres -h localhost -d payment_db -c "\dt"
# Harus muncul: payments

# notification_db
psql -U postgres -h localhost -d notification_db -c "\dt"
# Harus muncul: notifications_log
```

**Via pgAdmin / TablePlus:**
- Pilih database `order_db` → Schemas → public → Tables → harus ada `orders`, `order_items`, `saga_logs`
- Lakukan hal yang sama untuk 3 database lainnya

---

### Langkah 7 — Load Seed Data (Inventory)

```bash
# Via psql:
psql -U postgres -h localhost -d inventory_db \
  -f services/inventory-service/migrations/seed.sql

# Atau via Makefile (jika di Linux/Mac/WSL):
make seed
```

**Verifikasi:**
```sql
-- Di inventory_db
SELECT name, stock_available, price FROM products ORDER BY price DESC;
-- Harus muncul 10 produk dummy
```

---

### Rangkuman Checklist Validasi Fase 1

| # | Langkah | Ekspektasi |
|---|---------|-----------|
| 1 | `.env` diisi | File `.env` ada, `<user>` dan `<password>` sudah diganti |
| 2 | `create-databases.sql` dijalankan | 4 database muncul di `pg_database` |
| 3 | `docker compose up -d` | Container `order-nats` running, port 4222 & 8222 aktif |
| 4 | `migrate-all.sh` dijalankan | Semua 4 service menampilkan ✅ |
| 5 | `\dt` di setiap DB | Tabel-tabel sesuai PRD Section 6 muncul |
| 6 | Seed data | 10 produk muncul di `products` |

---

## Tech Stack

| Layer | Teknologi |
|---|---|
| Bahasa | Go 1.23+ |
| HTTP | Fiber v2 |
| gRPC | google.golang.org/grpc + protobuf |
| Message Broker | NATS JetStream |
| Database | PostgreSQL 14+ (per-service) |
| Migration | golang-migrate |
| Auth | golang-jwt/jwt v5 |
| Logging | zerolog |
| Metrics | prometheus/client_golang |

## NATS Event Subjects

| Subject | Publisher | Consumer |
|---|---|---|
| `order.created` | Order Service | Inventory Service |
| `stock.reserved` | Inventory Service | Order Service |
| `stock.rejected` | Inventory Service | Order Service |
| `payment.requested` | Order Service | Payment Service |
| `payment.success` | Payment Service | Order Service, Notification |
| `payment.failed` | Payment Service | Order Service, Notification |
| `order.cancelled` | Order Service | Inventory Service, Notification |
| `order.completed` | Order Service | Notification Service |

---

*Fase 2: Order Service CRUD + Auth JWT + Inventory Service gRPC*

## Menjalankan via Docker (Full Stack)

Untuk keperluan demo atau portofolio, seluruh backend (termasuk 4 instance PostgreSQL dan NATS) dapat dijalankan cukup dengan satu perintah melalui Docker Compose.

### Prasyarat
- Docker & Docker Compose terinstall dan berjalan (Docker Desktop).

### Langkah-langkah
1. **Persiapkan Environment Variables**
   Salin template environment untuk docker:
   ```bash
   cp .env.docker.example .env.docker
   ```

2. **Jalankan Docker Compose**
   Build dan jalankan seluruh container (berjalan di background):
   ```bash
   docker compose --env-file .env.docker up -d --build
   ```

3. **Verifikasi Container Sehat**
   Semua container database dan NATS memiliki _healthcheck_. Go services akan menunggu _healthcheck_ ini selesai (kondisi `service_healthy`) sebelum menyala.
   Cek status container:
   ```bash
   docker compose ps
   ```
   Pastikan keempat service (`order-service`, `inventory-service`, `payment-service`, `notification-service`) berstatus `Up`.

   Cek log service (contoh untuk order-service):
   ```bash
   docker compose logs -f order-service
   ```
   > **Catatan Auto-Migration & Auto-Seed**: Setiap service Go akan otomatis menjalankan migration `up` ke databasenya masing-masing saat startup. `inventory-service` juga otomatis mengisi seed produk (jika database masih kosong).

4. **Testing Sederhana (Create Order)**
   Setelah semua jalan, Anda bisa langsung mengetes API:
   ```bash
   # Register
   curl -X POST http://localhost:8080/api/v1/auth/register -H "Content-Type: application/json" -d '{"email": "docker@demo.com", "password": "123", "name": "Docker"}'
   
   # Login & Ambil Token (simpan di variable TOKEN)
   
   # Cek produk (Inventory gRPC call)
   curl -X GET http://localhost:8080/api/v1/products -H "Authorization: Bearer <TOKEN>"
   
   # Buka WebSocket di terminal terpisah
   wscat -c ws://localhost:8083/ws/orders/<ORDER_ID_YANG_AKAN_DIBUAT>
   
   # Create Order -> Trigger SAGA
   curl -X POST http://localhost:8080/api/v1/orders -H "Authorization: Bearer <TOKEN>" -H "Content-Type: application/json" -d '{"items": [{"product_id": "<PRODUCT_ID>", "quantity": 1, "price": 10000}]}'
   ```

5. **Stop & Cleanup**
   Untuk menghentikan semua service dan **menghapus seluruh data database** (hapus volume):
   ```bash
   docker compose down -v
   ```

