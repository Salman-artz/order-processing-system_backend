# Order Processing System - Backend

Backend event-driven untuk pemrosesan order e-commerce dengan pola **Saga Choreography**. Sistem terdiri dari beberapa microservice Go yang berkomunikasi melalui REST, gRPC, NATS JetStream, dan terintegrasi dengan Payment Gateway (Midtrans).

---

## Status & Fitur Project

Versi saat ini telah mengimplementasikan arsitektur mikroservis lengkap yang *production-ready*:

- **Order Service (:8080)**:
  - Register & Login JWT (Role: `CUSTOMER` vs `ADMIN`).
  - Pembuatan order & manajemen item order dengan resolusi katalog produk.
  - API Gateway / Reverse Proxy otomatis untuk `/products*` dan `/payments*`.
  - Saga Choreography consumer & publisher.
- **Inventory Service (:8081 & gRPC :50051)**:
  - **CRUD Produk Lengkap**: Create (`POST`), Update Harga & Nama (`PUT`), Delete (`DELETE`).
  - **Manajemen & Restock Stok**: Tambah stok (`PATCH /products/:id/stock` & `POST /products/:id/restock`).
  - **Saga Inventory Lock**: Reservasi stok (`stock.reserved`), pelepasan kompensasi (`order.cancelled`), dan konfirmasi permanen (`order.completed`).
  - Pemisahan status stok: **Available Stock** (tersedia) vs **Locked / Reserved Stock** (terkunci sementara oleh Saga).
  - gRPC server untuk query verifikasi cepat antar-service.
  - Auto-seed data produk awal saat database masih kosong.
- **Payment Service (:8082)**:
  - **Integrasi Payment Gateway Midtrans Snap**: Pembuatan token transaksi Snap (`/api/v1/payments/midtrans/snap`).
  - **Webhook Notification Callback**: Menerima callback status dari Midtrans (`/api/v1/payments/midtrans/notification`) dengan verifikasi digital signature **`SHA512(order_id + status_code + gross_amount + server_key)`**.
  - **Simulator Pembayaran**: Endpoint `/api/v1/payments/midtrans/simulate` untuk pengujian lokal (sukses/batal).
  - Order tertahan di `AWAITING_PAYMENT` sampai ada konfirmasi nyata dari payment gateway.
- **Notification Service (:8083 & WebSocket)**:
  - Pencatatan riwayat event saga ke `notification_db`.
  - Broadcast perubahan status order (`PENDING`, `RESERVED`, `AWAITING_PAYMENT`, `COMPLETED`, `CANCELLED`) ke browser secara real-time via WebSocket.
- **Database & Isolasi Data**:
  - **Database-per-Service Pattern**: 4 instance PostgreSQL terpisah (`order_db`, `inventory_db`, `payment_db`, `notification_db`).
  - Auto-migration saat service dimulai.
- **Docker & Testing**:
  - Docker Compose full-stack lokal.
  - Automated End-to-End (E2E) Test Suite Playwright (34 test cases passing).

---

## Arsitektur Sistem

```text
                         +-------------------+
 Customer / Admin ------>|   Order Service   | :8080 (API Gateway & Auth)
        REST             +---------+---------+
                                   │
                              NATS JetStream (:4222)
               ┌───────────────────┼───────────────────┐
               │                   │                   │
       +-------▼-------+   +-------▼-------+   +-------▼--------+
       |   Inventory   |   |    Payment    |   |  Notification  |
       | :8081 / gRPC  |   | :8082         |   | :8083 / WS     |
       +-------+-------+   +-------+-------+   +--------+-------+
               │                   │                    │
        inventory_db          payment_db          notification_db
         (Port 6433)          (Port 6434)           (Port 6435)

                         order_db (Port 6432)
```

---

## Alur Saga Choreography

```text
[Customer Checkout] ──► Order Service: Simpan "PENDING" & Publish "order.created"
                              │
                              ▼
[Inventory Service] ──► Cek Stok Barang
       ├─► (Stok Cukup) : Kurangi Available, Tambah Reserved & Publish "stock.reserved"
       │         │
       │         ▼
       │   [Order Service] : Ubah status jadi "AWAITING_PAYMENT" & Publish "payment.requested"
       │         │
       │         ▼
       │   [Payment Gateway / Midtrans] : Customer membayar via QRIS/VA/CC
       │         ├─► (Sukses / Settlement) :
       │         │       Publish "payment.success"
       │         │       └─► Order Service set "COMPLETED" & Inventory Service confirm stock
       │         │
       │         └─► (Gagal / Cancel / Expire) :
       │                 Publish "payment.failed"
       │                 └─► Order Service set "CANCELLED" & Inventory Service release stock (Rollback)
       │
       └─► (Stok Kurang) : Publish "stock.rejected"
                 └─► Order Service langsung set "CANCELLED"
```

---

## Tech Stack

| Area | Teknologi | Keterangan |
|---|---|---|
| **Bahasa** | Go 1.23+ | Native concurrency & high performance |
| **HTTP Framework** | Fiber v2 | REST routing & reverse proxy gateway |
| **RPC** | gRPC + Protocol Buffers v3 | Low-latency inter-service sync communication |
| **Event Broker** | NATS JetStream | Event-driven async messaging & stream persistence |
| **Payment Gateway** | Midtrans Snap & Webhook | Snap Token, SHA512 Signature verification |
| **Database** | PostgreSQL 16 | Isolated database-per-service pattern |
| **Migration** | golang-migrate | Automatic schema migrations on service startup |
| **Realtime** | WebSockets (gorilla) | Live order status push to client frontend |
| **Logging** | zerolog | Structured JSON logging |
| **Testing** | Playwright & Go testing | 34 automated E2E tests covering full lifecycle |
| **Containerization** | Docker & Docker Compose v2 | Local multi-container orchestration |

---

## Struktur Repository

```text
services/
  order-service/         # API order, auth JWT, reverse proxy gateway, saga consumer
  inventory-service/     # Produk CRUD, restock, saga lock/release, REST & gRPC
  payment-service/       # Integrasi Midtrans Snap, webhook callback, SHA512, simulation
  notification-service/  # Event audit log & WebSocket push server
pkg/
  events/                # NATS Stream subjects dan payload schemas
  jwtutil/               # Helper pembuatan & verifikasi JWT
  logger/                # Shared zerolog logger
  natsclient/            # Shared NATS JetStream client connector
  postgres/              # PostgreSQL connection pool & migration runner
proto/                   # Definisi gRPC Protocol Buffers inventory
scripts/                 # Database creation & migration scripts
docker-compose.yml       # Full stack local docker compose
Makefile                 # Command shorthand development
```

---

## Menjalankan dengan Docker Compose

1. Siapkan file environment:
   ```bash
   cp .env.docker.example .env.docker
   ```

2. Jalankan seluruh stack (NATS, 4 Database PostgreSQL, dan 4 Microservices Go):
   ```bash
   docker compose --env-file .env.docker up -d --build
   ```

3. Cek status container:
   ```bash
   docker compose ps
   ```

---

## Port & Akses Layanan

| Komponen / Service | URL / Port | Keterangan |
|---|---|---|
| **Order Service (API Gateway)** | `http://localhost:8080` | Endpoint utama customer, admin, & reverse proxy |
| **Inventory Service (REST)** | `http://localhost:8081` | REST catalog & stock management |
| **Inventory Service (gRPC)** | `localhost:50051` | gRPC sync stock verification |
| **Payment Service** | `http://localhost:8082` | Midtrans Snap & Webhook endpoints |
| **Notification Service** | `ws://localhost:8083/ws/orders/:id` | WebSocket real-time broadcast |
| **NATS Broker** | `nats://localhost:4222` | JetStream message broker |
| **NATS Monitoring** | `http://localhost:8222` | NATS web monitoring dashboard |

### Akses Database PostgreSQL:

Semua database berjalan dengan user `postgres` dan password `123`:

| Database | Port Host | Nama DB |
|---|---|---|
| **Order DB** | `localhost:6432` | `order_db` |
| **Inventory DB** | `localhost:6433` | `inventory_db` |
| **Payment DB** | `localhost:6434` | `payment_db` |
| **Notification DB** | `localhost:6435` | `notification_db` |

Akses CLI langsung via Docker:
```bash
docker exec -it postgres-order psql -U postgres -d order_db
docker exec -it postgres-inventory psql -U postgres -d inventory_db
docker exec -it postgres-payment psql -U postgres -d payment_db
```

---

## Daftar Endpoint API Utama

### 1. Autentikasi
- `POST /api/v1/auth/register` : Registrasi akun baru (Customer / Admin).
- `POST /api/v1/auth/login` : Login akun & mendapatkan JWT token.

### 2. Pesanan (Customer)
- `POST /api/v1/orders` : Membuat pesanan baru (Memicu alur Saga).
- `GET /api/v1/orders` : Menampilkan daftar pesanan milik customer.
- `GET /api/v1/orders/:id` : Menampilkan detail pesanan & item produk.

### 3. Monitoring (Admin)
- `GET /api/v1/admin/orders` : Menampilkan seluruh pesanan di sistem.
- `GET /api/v1/admin/orders/:id/timeline` : Menampilkan log audit trail tahapan Saga.

### 4. Inventaris & Produk (Admin / Public)
- `GET /api/v1/products` : Menampilkan seluruh katalog produk dan ketersediaan stok.
- `POST /api/v1/products` : Menambah produk baru ke katalog.
- `PUT /api/v1/products/:id` : Mengedit nama atau harga produk.
- `PATCH /api/v1/products/:id/stock` : Mengisi ulang (*restock*) stok barang.
- `POST /api/v1/products/:id/restock` : Restock endpoint alias.
- `DELETE /api/v1/products/:id` : Menghapus produk dari inventaris.

### 5. Pembayaran (Payment Gateway / Midtrans)
- `POST /api/v1/payments/midtrans/snap` : Menghasilkan Snap Token transaksi Midtrans.
- `POST /api/v1/payments/midtrans/notification` : Webhook callback resmi Midtrans (Verifikasi signature SHA512).
- `POST /api/v1/payments/midtrans/simulate` : Simulasi pembayaran lokal (*settlement* / *cancel*).

---

## Event NATS JetStream

| Subject | Publisher | Consumer | Deskripsi |
|---|---|---|---|
| `order.created` | Order Service | Inventory, Notification | Pesanan dibuat, memicu pengecekan stok |
| `stock.reserved` | Inventory Service | Order Service, Notification | Stok berhasil dikunci untuk order |
| `stock.rejected` | Inventory Service | Order Service, Notification | Stok tidak mencukupi, pesanan dibatalkan |
| `payment.requested` | Order Service | Payment Service | Permintaan proses pembayaran |
| `payment.success` | Payment Service | Order Service, Notification | Pembayaran terverifikasi lunas |
| `payment.failed` | Payment Service | Order Service, Notification | Pembayaran ditolak/dibatalkan |
| `order.completed` | Order Service | Inventory, Notification | Pesanan selesai, stok dipotong permanen |
| `order.cancelled` | Order Service | Inventory, Notification | Pesanan dibatalkan, stok di-rollback ke Available |
