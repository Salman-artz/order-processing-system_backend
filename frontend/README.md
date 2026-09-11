# Order Processing System — Frontend

A **Next.js 14+** (App Router) web application for the Order Processing System backend microservices. Provides a customer-facing catalog & order flow, plus an admin dashboard with real-time Saga Orchestration visualization.

**Backend repo:** [`../order-processing-system`](../order-processing-system) — runs independently on `localhost:8080`.

---

## Tech Stack

| Layer | Library |
|---|---|
| Framework | Next.js 16 (App Router, TypeScript) |
| Styling | Tailwind CSS v4 |
| Data Fetching | TanStack Query (React Query) |
| HTTP Client | Axios |
| State | Zustand |
| Forms | react-hook-form + Zod |
| Real-time | Native WebSocket (exponential-backoff reconnect) |
| Toasts | Sonner |
| Charts | Recharts |
| Animation | Framer Motion |

---

## Quick Start (Local Development)

**Prerequisites:** Node.js 20+, backend running on `localhost:8080`

```bash
# 1. Install dependencies
npm install

# 2. Copy environment file
cp .env.local.example .env.local

# 3. Start dev server
npm run dev
```

Open [http://localhost:3000](http://localhost:3000).

**Demo credentials (mock mode):**
- Customer: any email + any password ≥ 8 chars
- Admin: email containing `admin` + any password ≥ 8 chars

---

## Menjalankan via Docker

### Prasyarat

- Docker Desktop (Windows/Mac) **atau** Docker Engine + Docker Compose v2 (Linux)
- Backend sudah berjalan dan bisa diakses di **`localhost:8080`** pada host machine
  - Jika backend berjalan native: pastikan Fiber/HTTP server bind ke `0.0.0.0:8080`, bukan `127.0.0.1:8080`
  - Jika backend berjalan via Docker: pastikan port 8080 sudah di-expose ke host (`ports: - "8080:8080"` di docker-compose backend)

### Langkah

```bash
# 1. Copy environment file dan sesuaikan jika perlu
cp .env.docker.example .env.docker

# 2. Build image dan jalankan container
docker compose --env-file .env.docker up -d --build

# 3. Pantau log startup
docker compose logs -f frontend
```

### Akses Aplikasi

Buka [http://localhost:3000](http://localhost:3000) di browser.

Container melakukan healthcheck ke `http://localhost:3000` setiap 30 detik. Cek status:
```bash
docker compose ps
# frontend container harus berstatus "healthy" atau "Up"
```

### Mengganti Port atau URL Backend

Edit `.env.docker` sebelum menjalankan `docker compose up`:

```dotenv
# Contoh: backend sudah di-deploy ke server lain
NEXT_PUBLIC_API_BASE_URL=https://api.production-domain.com/api/v1
NEXT_PUBLIC_WS_URL=wss://api.production-domain.com/ws

# Contoh: ganti port frontend jika 3000 sudah dipakai
FRONTEND_PORT=8090
```

> ⚠️ **Penting — Build Time Embedding:** Variabel `NEXT_PUBLIC_*` dikompilasi ke dalam JavaScript bundle oleh `next build` saat **build time**, bukan runtime. Jika Anda mengubah nilai di `.env.docker`, Anda **harus rebuild image**:
> ```bash
> docker compose --env-file .env.docker up -d --build
> ```

### Menghentikan Container

```bash
docker compose down
```

---

## Troubleshooting Docker

### Frontend tidak bisa connect ke backend (connection refused / ERR_CONNECTION_REFUSED)

1. **Cek apakah backend benar-benar accessible dari host:**
   ```bash
   curl http://localhost:8080/health
   # Harus return 200 OK
   ```

2. **Jika backend jalan native (bukan Docker):** pastikan server bind ke `0.0.0.0`, bukan `127.0.0.1`. Di dalam container Docker, `host.docker.internal` memang resolve ke IP host, tapi koneksi tetap akan ditolak kalau server hanya listen di `127.0.0.1` (loopback).

3. **Jika backend jalan via Docker (compose project terpisah):** pastikan `ports:` backend sudah di-expose ke host:
   ```yaml
   # di docker-compose backend
   services:
     order-service:
       ports:
         - "8080:8080"   # ← wajib ada supaya host.docker.internal bisa reach
   ```

4. **Linux — `host.docker.internal` tidak dikenal:** `docker-compose.yml` frontend sudah include `extra_hosts: host.docker.internal:host-gateway`. Jika masih gagal, coba:
   ```bash
   # Temukan IP gateway Docker
   docker inspect ops-frontend | grep Gateway
   # Lalu set manual di .env.docker
   NEXT_PUBLIC_API_BASE_URL=http://<gateway-ip>:8080/api/v1
   ```

5. **CORS error di browser console:** Backend perlu mengizinkan origin `http://localhost:3000`. Cek konfigurasi CORS di order-service (Fiber middleware cors).

### Image terlalu besar

Build menggunakan `output: 'standalone'` di `next.config.ts` dan multi-stage Dockerfile, sehingga image runtime tidak mengandung `node_modules` (hanya file yang dibutuhkan untuk serve). Image runner biasanya ~150-200MB.

```bash
docker images ops-frontend
```

### Melihat variabel env yang ter-embed di bundle

```bash
# Inspect nilai NEXT_PUBLIC_ yang sudah di-bake saat build
docker compose exec frontend env | grep NEXT
# Catatan: nilai ini sudah di-bake ke JS, bukan berasal dari env runtime container
```

---

## Struktur Folder

```
frontend/
├── app/
│   ├── (auth)/
│   │   ├── login/          # Halaman login
│   │   └── register/       # Halaman register
│   ├── admin/
│   │   ├── dashboard/      # Admin dashboard (charts, KPIs)
│   │   └── orders/         # List + detail order (Saga timeline)
│   ├── customer/
│   │   ├── orders/         # Riwayat + detail order (stepper real-time)
│   │   └── products/       # Katalog produk + form order
│   ├── globals.css
│   ├── layout.tsx
│   └── providers.tsx
├── components/
│   ├── SagaTimeline.tsx    # Saga visualizer (horizontal stepper + compact)
│   └── Sidebar.tsx         # Fixed left navigation
├── lib/
│   ├── api/
│   │   └── client.ts       # Axios instance + auth interceptor
│   ├── hooks/
│   │   └── useOrderSocket.ts  # WebSocket hook (exponential backoff)
│   └── store/
│       └── useAuthStore.ts # Zustand auth state
├── types/
│   └── index.ts            # TypeScript interfaces
├── Dockerfile              # Multi-stage build (deps → builder → runner)
├── docker-compose.yml      # Frontend-only service
├── .env.docker.example     # Template env untuk Docker deployment
├── .env.local.example      # Template env untuk local dev
└── next.config.ts          # output: 'standalone'
```

---

## Environment Variables

| Variable | Deskripsi | Default |
|---|---|---|
| `NEXT_PUBLIC_API_BASE_URL` | Base URL REST API backend | `http://localhost:8080/api/v1` |
| `NEXT_PUBLIC_WS_URL` | WebSocket URL untuk real-time update | `ws://localhost:8080/ws` |
| `NEXT_PUBLIC_USE_MOCK` | Gunakan mock data (`true`/`false`) | `false` |
| `FRONTEND_PORT` | Port host untuk Docker mapping | `3000` |
