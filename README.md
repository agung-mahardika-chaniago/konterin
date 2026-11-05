# FixPhone — Smartphone Repair Management System

Aplikasi manajemen usaha servis HP untuk teknisi, toko servis, dan usaha reparasi.
Dirancang sebagai **modular monolith** yang siap dikembangkan menjadi produk SaaS
multi-tenant dan multi-outlet.

> **Status proyek: Tahap 5 — Modul Penerimaan Servis, Pelanggan, Foto & Label (selesai).**
> Tahap 6 (diagnosis, pengerjaan, status machine, item jasa/part) berikutnya.
> Rincian implementasi: [`docs/12-implementasi-tahap-5.md`](docs/12-implementasi-tahap-5.md).

---

## 1. Ringkasan Teknologi

| Lapisan | Teknologi |
|---|---|
| Backend | Go 1.24+, Gin, GORM |
| Database | PostgreSQL 16 |
| Auth | JWT HS256 + refresh token rotasi (cookie HttpOnly) |
| Frontend | React 18.3 + TypeScript 5.7 + Vite 5 |
| UI | Tailwind CSS 3.4 + komponen lokal kompatibel shadcn/ui + Lucide |
| State/data | Zustand (sesi & outlet); TanStack Query, React Hook Form + Zod direncanakan Tahap 6+ |
| Grafik | Recharts |
| Cetak | Invoice A4, struk thermal 58 mm / 80 mm, label servis |
| Deployment | Docker Compose + Nginx |
| PWA | Manifest + service worker (app shell) |

## 2. Struktur Repository

```
.
├── backend/          # API Go (Gin + GORM)
├── frontend/         # SPA React + TypeScript
├── deploy/           # Docker, Nginx, backup & restore
├── docs/             # Analisis, arsitektur, database, API, UI/UX
├── scripts/          # Helper development
├── docker-compose.yml
└── README.md
```

Penjelasan lengkap: [`docs/04-struktur-folder.md`](docs/04-struktur-folder.md).

## 3. Dokumen Proyek

| Dokumen | Isi |
|---|---|
| [01 — Analisis Kebutuhan](docs/01-analisis-kebutuhan.md) | Aktor, kebutuhan fungsional, aturan bisnis, NFR, risiko |
| [02 — Arsitektur Sistem](docs/02-arsitektur.md) | Context/container, layer, multi-tenancy, auth, deployment, ADR |
| [03 — Desain Database](docs/03-desain-database.md) | ERD, kamus tabel, index, transaksi, enkripsi, seed |
| [04 — Struktur Folder](docs/04-struktur-folder.md) | Struktur backend/frontend, konvensi kode & penamaan |
| [05 — Roadmap MVP](docs/05-roadmap-mvp.md) | 12 tahap, deliverable, acceptance criteria, testing gate |
| [06 — Kontrak API](docs/06-kontrak-api.md) | Konvensi REST, envelope, error code, daftar endpoint |
| [07 — Keamanan](docs/07-keamanan.md) | Threat model, RBAC, secrets, audit, backup, privasi |
| [08 — UI/UX Guideline](docs/08-uiux-guideline.md) | Design token, layout, komponen, status badge, print, PWA |
| [09 — Implementasi Tahap 2](docs/09-implementasi-tahap-2.md) | Migrasi, seed, environment, portable DB, CLI, health, hasil test |
| [10 — Implementasi Tahap 3](docs/10-implementasi-tahap-3.md) | Auth JWT, refresh rotation, RBAC, user management, frontend login, hasil test |
| [11 — Implementasi Tahap 4](docs/11-implementasi-tahap-4.md) | App shell, dashboard data nyata, grafik, alert, PWA, hasil test |
| [12 — Implementasi Tahap 5](docs/12-implementasi-tahap-5.md) | Modul servis intake, pelanggan, kredensial terenkripsi, foto, cetak label |
| [OpenAPI Tahap 5](docs/openapi.yaml) | Kontrak endpoint auth, user, role, outlet, dashboard, customer, servis, health |

## 4. Cara Menjalankan (Development)

### 4.1 Database

Pilih salah satu. **PostgreSQL portable** dipakai bila Docker tidak tersedia.

```powershell
# Opsi A — portable (tanpa Docker/admin), PostgreSQL 16.15 di 127.0.0.1:55432
powershell -ExecutionPolicy Bypass -File scripts/dev-db.ps1 start

# Opsi B — Docker, PostgreSQL 16 di 127.0.0.1:55432 (DSN sama)
docker compose up -d
```

### 4.2 Environment & backend

```powershell
# 1. Salin environment lalu ganti JWT_SECRET, APP_ENC_KEY, dan password seed
Copy-Item .env.example .env

# 2. Buat database dev + test dan jalankan migrasi
cd backend
go run ./cmd/cli db-setup
go run ./cmd/cli migrate up

# 3. Seed role/permission/super admin (aman diulang)
go run ./cmd/cli seed

# 4. Buat tenant pertama (owner + outlet + pengaturan)
go run ./cmd/cli create-tenant `
  --name "Toko Demo" --slug toko-demo `
  --owner-name "Budi" --owner-email "owner@toko.id" --owner-password "Rahasia123" `
  --outlet-name "Outlet Utama" --outlet-code MAIN

# 5. Jalankan API
go run ./cmd/api
```

Health & meta:

- `http://localhost:8080/healthz` — proses hidup
- `http://localhost:8080/readyz` — database + versi migrasi
- `http://localhost:8080/version` — metadata build

Test dan pemeriksaan:

```powershell
cd backend
go vet ./...
go test ./... -count=1
```

Perintah lain: `go run ./cmd/cli migrate status`, `migrate down --steps 1`,
`db-drop --name fixphone_test --yes`. Ringkasan lengkap ada di
[`docs/09-implementasi-tahap-2.md`](docs/09-implementasi-tahap-2.md).

### 4.3 Frontend

```powershell
cd frontend
Copy-Item .env.example .env   # opsional; default memakai proxy Vite
npm install
npm run dev                   # http://127.0.0.1:5173
```

Vite mem-proxy `/api` ke `http://127.0.0.1:8080`. Buka
`http://127.0.0.1:5173`, lalu masuk dengan akun owner hasil `create-tenant`.
Sesi bertahan saat halaman dimuat ulang tanpa menyimpan access token di
`localStorage`. Setelah login, dashboard menampilkan angka, grafik 7/30 hari,
peringatan, dan aktivitas teknisi dari database. Outlet dapat diganti dari
switcher di topbar (multi-outlet).

Pemeriksaan frontend:

```powershell
cd frontend
npm run typecheck
npm run build
```

Perilaku autentikasi, RBAC, dan cara reset password dijelaskan di
[`docs/10-implementasi-tahap-3.md`](docs/10-implementasi-tahap-3.md).

### 4.4 Makefile

Jika GNU Make tersedia:

```bash
make setup        # start DB portable + buat database + unduh dependency
make migrate-up
make seed
make test
make api
```

## 5. Prasyarat Development

- Go **1.24+**
- PostgreSQL **16+** — portable binary disediakan (`scripts/dev-db.ps1`)
- Node.js **20+** dan npm **10+** (wajib untuk frontend)
- Docker Desktop **opsional** untuk parity dengan production

## 6. Konvensi Utama

- Bahasa kode: **Inggris**; bahasa UI & pesan error: **Indonesia**.
- Semua waktu disimpan **UTC** (`timestamptz`), ditampilkan dalam zona waktu outlet
  (default `Asia/Jakarta`).
- Semua nominal uang disimpan sebagai **BIGINT rupiah** (tanpa desimal).
- Setiap data transaksional wajib memiliki `tenant_id` dan `outlet_id`.
- Tidak ada data dummy di aplikasi produksi; seed hanya berisi data sistem
  (role, permission, super admin, outlet default).
- Setiap tombol di UI harus terhubung ke API nyata atau disembunyikan.

## 7. Lisensi

Proprietary — © 2026 FixPhone. Seluruh hak dilindungi.
