# Order Processing System - Distributed Microservices (Backend)

Backend berkinerja tinggi (*High-Throughput Distributed Microservices*) untuk sistem pemrosesan pesanan terdistribusi dengan pola **Saga Choreography**. Dibangun menggunakan Go (Golang), Fiber, NATS JetStream, PostgreSQL (*Database-per-Service*), gRPC, Docker Compose, dan terintegrasi dengan Payment Gateway Midtrans.

---

## 📸 Antarmuka Observabilitas & Real Distributed Execution Preview

| Preview Antarmuka | Deskripsi Fitur & Rekayasa Sistem |
|---|---|
| <img src="screenshots/live_real_step2_order_commit_nats.png" alt="Real Order Commit Flow" width="540" /> | **Live Order Commit Flow (NATS JetStream)**<br/>Transaksi nyata multi-service: `order-service` mendispatch `order.created`, `inventory-service` mengunci stok di PostgreSQL, dan `payment-service` memproses pembayaran hingga status `COMPLETED` dalam 26ms. |
| <img src="screenshots/live_real_step3_saga_rollback_nats.png" alt="Real Saga Compensatory Rollback" width="540" /> | **Real-Time Saga Compensatory Rollback**<br/>Simulasi kegagalan pembayaran: NATS mempublikasikan `payment.failed`, memicu transaksi kompensasi otomatis pada `inventory-service` untuk merestore stok (*lock released*) dan membatalkan pesanan. |
| <img src="screenshots/live_real_step3_trace_rollback_drawer.png" alt="Distributed Trace Waterfall & JSON Payload" width="540" /> | **Distributed Trace Waterfall & Raw JSON Payload**<br/>Drawer telemetri interaktif untuk inspeksi rincian span latensi per mikroservis, trace ID, correlation ID, dan payload event mentah NATS JetStream. |
| <img src="screenshots/live_real_step4_orders_registry.png" alt="Distributed Orders Registry" width="540" /> | **Distributed Orders Registry (PostgreSQL Audit)**<br/>Audit log database persisten lintas transaksi terdistribusi yang mencatat status final `COMPLETED` dan `CANCELLED` secara deterministic. |
| <img src="screenshots/live_real_step5_order_saga_timeline.png" alt="Saga Execution Timeline UI" width="540" /> | **Saga Execution Timeline UI**<br/>Visualisasi alur langkah transaksi terdistribusi, durasi latensi per span, serta penanganan kompensasi rollback inventaris otomatis. |
| <img src="screenshots/live_real_step1_inventory_restock.png" alt="PostgreSQL Inventory Ledger" width="540" /> | **PostgreSQL Inventory Ledger & Stock Isolation**<br/>Pemisahan *Available Stock* vs *Reserved Stock (Saga Locks)* dengan penanganan race condition berbasis transaksi atomik PostgreSQL. |
| <img src="screenshots/live_real_step7_telemetry_dashboard.png" alt="Real-time Order Monitoring Dashboard" width="540" /> | **Real-time Order Monitoring Dashboard**<br/>Konsol observabilitas terpusat: status 4 microservice, metrik throughput event NATS JetStream, dan rasio keberhasilan Saga. |
| <img src="screenshots/foto1_login_light_mode.png" alt="Login Mode Terang" width="540" /> | **Login Mode Terang (Light Mode)**<br/>Antarmuka berstandar enterprise dengan palet Titanium Canvas dan tipografi monospace berdensitas tinggi. |
| <img src="screenshots/foto2_login_dark_mode.png" alt="Login Mode Gelap" width="540" /> | **Login Mode Gelap (Dark Mode)**<br/>Palet Titanium Gunmetal & Industrial Amber dengan rasio kontras tinggi dan telemetry metadata status cluster. |

---

## 🏗️ Clean Architecture & Struktur Modul Backend

Backend menerapkan prinsip **Clean Architecture & Separation of Concerns** pada setiap microservice:
1. **Presentation Layer (`internal/handler`)**: Fiber HTTP handlers, router endpoints, reverse-proxy API gateway, middleware autentikasi JWT stateless, dan CORS handler.
2. **Application Layer (`internal/service`)**: Domain business logic, Saga orchestration flow, kalkulasi saldo, dan validasi transaksi.
3. **Domain Layer (`pkg/events`, `internal/repository/models.go`)**: Entitas inti, kontrak event payload, subjek NATS JetStream, dan model status transaksi.
4. **Infrastructure Layer (`internal/repository`, `pkg/natsclient`, `pkg/postgres`)**: Implementasi akses database PostgreSQL via connection pool, query SQLC, dan pub/sub NATS JetStream client.

---

## ⚡ Alur Saga Choreography & Kompensasi Rollback

```mermaid
sequenceDiagram
    autonumber
    participant Client as Web Client (Next.js)
    participant Order as Order Service (:8085)
    participant NATS as NATS JetStream (:4222)
    participant Inv as Inventory Service (:8081)
    participant Pay as Payment Service (:8082)

    Client->>Order: POST /api/v1/orders (Place Order)
    Order->>Order: Simpan Order (Status: PENDING)
    Order->>NATS: Publish (order.created)
    
    NATS->>Inv: Consume (order.created)
    Note over Inv: Lock stok di PostgreSQL (stock_reserved)
    alt Stok Cukup
        Inv->>NATS: Publish (inventory.reserved)
    else Stok Habis
        Inv->>NATS: Publish (inventory.failed)
    end

    NATS->>Pay: Consume (inventory.reserved)
    Note over Pay: Verifikasi Pembayaran (Midtrans / Sandbox)
    alt Pembayaran Berhasil (Commit Flow)
        Pay->>NATS: Publish (payment.success)
        NATS->>Order: Consume (payment.success) ➔ Status: COMPLETED
        NATS->>Inv: Consume (order.completed) ➔ Finalisasi Pengurangan Stok
    else Pembayaran Gagal (Rollback Flow)
        Pay->>NATS: Publish (payment.failed)
        NATS->>Inv: Consume (payment.failed) ➔ SAGA KOMPENSASI
        Note over Inv: Buka kunci stok (stock_reserved -> stock_available)
        NATS->>Order: Consume (order.cancelled) ➔ Status: CANCELLED
    end
```

---

## 🛠️ Tech Stack & Spesifikasi

- **Bahasa & Framework**: Go (Golang 1.22+), Fiber v2
- **Message Broker**: NATS JetStream (Pub/Sub terjamin, At-least-once delivery, Deduplication)
- **Database**: PostgreSQL 16 dengan pola **Database-per-Service** (4 Database terisolasi: `order_db` :7432, `inventory_db` :7433, `payment_db` :7434, `notification_db` :7435)
- **Protokol Komunikasi**: REST API (HTTP/2), gRPC (High-speed inter-service query), Protocol Buffers
- **Observabilitas**: OpenTelemetry Distributed Tracing spans, Structured JSON Logging (zerolog)
- **Containerization**: Docker & Docker Compose (9 active containers)

---

## 🚀 Menjalankan Backend Secara Lokal (Real Microservices Stack)

```bash
# 1. Jalankan seluruh container stack (NATS + 4 PostgreSQL + 4 Go Microservices)
docker compose up -d

# 2. Periksa status container
docker compose ps

# 3. Jalankan unit & integration test microservices
cd services/order-service && go test ./...
cd ../inventory-service && go test ./...
cd ../payment-service && go test ./...
cd ../notification-service && go test ./...
```
