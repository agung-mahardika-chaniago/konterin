# 09 — Implementasi Tahap 2: Database & Environment

**Status:** selesai · **Versi aplikasi:** 0.2.0 · **Tanggal:** 3 November 2026
**Acuan:** [`05 — Roadmap MVP`](05-roadmap-mvp.md) Tahap 2, [`03 — Desain Database`](03-desain-database.md), [`07 — Keamanan`](07-keamanan.md)

---

## 1. Tujuan Tahap 2

Tahap 2 membuktikan bahwa desain database Tahap 1 dapat dijalankan sebagai
database PostgreSQL nyata, dikelola dengan migrasi yang reproducible, di-seed
secara idempoten, dan dapat diakses aplikasi melalui konfigurasi environment.

Yang dihasilkan:

1. 15 pasang migrasi SQL (up/down) untuk 31 tabel.
2. CLI `migrate`, `seed`, `create-tenant`, `db-create`, `db-drop`, `db-setup`.
3. Koneksi GORM + pgx dengan connection pool, ping, dan graceful shutdown.
4. Environment tervalidasi (`.env.example`, `.env` lokal yang diabaikan Git).
5. PostgreSQL 16.15 portable untuk development tanpa Docker/admin.
6. Docker Compose PostgreSQL 16 untuk parity deployment.
7. Test integrasi: migrasi up → down → up, constraint kritis, seed idempoten,
   provisioning tenant, dan endpoint health.
8. Endpoint `GET /healthz`, `GET /readyz`, `GET /version` (plus alias `/api/v1`).

---

## 2. File yang Ditambahkan

### Backend

```
backend/
├── go.mod, go.sum
├── cmd/
│   ├── api/main.go                  # HTTP server + graceful shutdown
│   └── cli/main.go                  # migrate, seed, create-tenant, db-*
├── internal/
│   ├── config/config.go             # load .env, validasi, LoadDBOnly
│   ├── config/config_test.go
│   ├── database/
│   │   ├── postgres.go              # pool, ping, close
│   │   ├── migrate.go               # runner embedded + ExpectedVersion
│   │   ├── seed.go                  # role, permission, mapping, super admin
│   │   ├── tenant.go                # provisioning tenant + outlet + owner
│   │   ├── dbadmin.go               # create/drop database
│   │   └── validation.go            # email & password policy
│   ├── pkg/logger/logger.go         # slog JSON
│   ├── pkg/version/version.go       # metadata build
│   └── router/router.go             # middleware + health/meta routes
├── migrations/
│   ├── 000001_extensions.{up,down}.sql
│   ├── 000002_tenants_outlets.{up,down}.sql
│   ├── 000003_roles_permissions_users.{up,down}.sql
│   ├── 000004_refresh_tokens.{up,down}.sql
│   ├── 000005_business_settings_sequences.{up,down}.sql
│   ├── 000006_customers_devices.{up,down}.sql
│   ├── 000007_spareparts_inventory.{up,down}.sql
│   ├── 000008_service_orders.{up,down}.sql
│   ├── 000009_service_diagnostics_history_photos.{up,down}.sql
│   ├── 000010_invoices_payments.{up,down}.sql
│   ├── 000011_warranties_claims.{up,down}.sql
│   ├── 000012_expenses.{up,down}.sql
│   ├── 000013_notifications_whatsapp.{up,down}.sql
│   ├── 000014_audit_logs.{up,down}.sql
│   ├── 000015_indexes_constraints.{up,down}.sql
│   └── migrations.go                # //go:embed *.sql
└── tests/
    ├── testutil/db.go               # helper DB test
    └── integration/
        ├── main_test.go
        ├── schema_test.go
        ├── seed_test.go
        └── health_test.go
```

### Root, deploy, dan script

```
.env.example
docker-compose.yml
docker-compose.override.yml
Makefile
deploy/docker/postgres/init.sql
scripts/dev-db.ps1
docs/09-implementasi-tahap-2.md
```

---

## 3. Cakupan Database

- **31 tabel** dibuat oleh 15 migrasi. Tabel `schema_migrations` milik
  golang-migrate tidak dihitung.
- **17 trigger** `updated_at` otomatis pada tabel yang memilikinya.
- **Index kritis** termasuk:
  - `uq_customers_tenant_phone_active` (unik per tenant, hanya baris aktif),
  - `idx_customers_name_trgm` (pencarian nama dengan `pg_trgm`),
  - `idx_spareparts_low_stock` (partial index stok minimum),
  - index antrean WhatsApp, invoice/payment, dan audit log.
- **Constraint penting** yang diuji:
  - stok sparepart tidak boleh negatif,
  - status service order hanya nilai yang terdaftar,
  - pasangan `credential_type`/`credential_enc` konsisten,
  - `(tenant_id, ticket_no)` unik, tetapi tiket yang sama boleh ada di tenant lain,
  - nomor HP pelanggan unik per tenant untuk baris aktif,
  - `paid_amount <= total_amount`,
  - `end_date >= start_date` pada garansi,
  - `audit_logs` tidak memiliki `updated_at` (append-only).
- `business_settings` memakai `UNIQUE NULLS NOT DISTINCT (tenant_id, outlet_id)`
  sehingga satu tenant hanya punya satu pengaturan global (`outlet_id NULL`).
- Enum memakai `VARCHAR + CHECK` agar evolusi nilai lebih mudah.
- Uang memakai `BIGINT` rupiah; kuantitas jasa `NUMERIC(12,2)`.

RLS (Row Level Security) **belum** diaktifkan pada Tahap 2. RLS dijadwalkan pada
Tahap 11 sesuai roadmap; saat ini isolasi tenant dijaga oleh `tenant_id` di setiap
query dan constraint database.

---

## 4. Seed Sistem

`cli seed` bersifat idempoten dan dapat dijalankan berulang.

| Data | Jumlah |
|---|---:|
| Role sistem | 5 (`super_admin`, `owner`, `admin`, `technician`, `cashier`) |
| Permission | 48 |
| Mapping role-permission | 123 |
| Super admin | 1 (dari environment) |
| Kategori pengeluaran per tenant | 6 |

Mapping yang dipakai:

- `owner` — seluruh 46 permission tenant.
- `admin` — 42 permission; tanpa `service.credential.reveal`,
  `service.override_unpaid_pickup`, `report.profit.view`, dan `audit.view`.
- `technician` — 14 permission.
- `cashier` — 19 permission.
- `super_admin` — 2 permission platform.

Password disimpan dengan bcrypt cost **12**. Seed super admin memakai
`SEED_SUPERADMIN_EMAIL` dan `SEED_SUPERADMIN_PASSWORD`, serta menandai
`must_change_password = true`.

`cli create-tenant` membuat dalam satu transaksi:

1. tenant (`status=active`, `plan=free`),
2. outlet utama,
3. pengaturan bisnis global,
4. akun owner dengan role `owner`,
5. enam kategori pengeluaran default.

Slug tenant, email owner, dan email pengguna lain diperiksa lebih dulu sehingga
tidak menghasilkan error constraint yang membingungkan.

---

## 5. Environment

Salin template lalu ganti rahasia:

```powershell
Copy-Item .env.example .env
```

Variabel wajib untuk API:

| Variabel | Aturan |
|---|---|
| `DB_DSN` | URL `postgres://` atau keyword DSN dengan `host` dan `dbname` |
| `JWT_SECRET` | minimal 32 karakter |
| `APP_ENC_KEY` | base64 dari tepat 32 byte |
| `APP_ENV` | `development`, `test`, atau `production` |
| `APP_PORT` | angka |

Variabel operasional lain: `CORS_ORIGINS`, `UPLOAD_DIR`, `UPLOAD_MAX_MB`,
`DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_LIFETIME`,
`DB_CONN_MAX_IDLE_TIME`, `LOG_LEVEL`.

CLI `migrate`, `seed`, `create-tenant`, dan `db-*` hanya membutuhkan `DB_DSN`
(memakai `LoadDBOnly`), sehingga database bisa disiapkan sebelum secret API diisi.
File `.env` diabaikan Git; `.env.example` tidak.

Contoh menghasilkan kunci:

```bash
# APP_ENC_KEY: 32 byte acak dalam base64
openssl rand -base64 32

# JWT_SECRET: 48 byte acak
openssl rand -base64 48
```

---

## 6. PostgreSQL Portable (Development)

Karena Docker tidak tersedia di mesin pengembangan, disediakan PostgreSQL 16.15
portable dari binary Zonky embedded-postgres. Binary dan data berada di `.dev/`
(diabaikan Git).

```powershell
# Unduh + ekstrak binary (sekali saja)
powershell -ExecutionPolicy Bypass -File scripts/dev-db.ps1 download

# Inisialisasi cluster + jalankan di 127.0.0.1:55432
powershell -ExecutionPolicy Bypass -File scripts/dev-db.ps1 start

# Cek status
powershell -ExecutionPolicy Bypass -File scripts/dev-db.ps1 status

# Hentikan
powershell -ExecutionPolicy Bypass -File scripts/dev-db.ps1 stop

# Hapus data dev lalu inisialisasi ulang (destruktif)
powershell -ExecutionPolicy Bypass -File scripts/dev-db.ps1 reset
```

Konfigurasi cluster:

- superuser `postgres`, autentikasi `trust` (hanya `listen_addresses=127.0.0.1`),
- port `55432`,
- data `.dev/pgdata`, log `.dev/postgres.log`.

`trust` hanya untuk development lokal. Produksi wajib memakai password/TLS
melalui DSN terkelola.

---

## 7. Perintah Harian

```powershell
# 1. Siapkan database dev + test
cd backend
go run ./cmd/cli db-setup

# 2. Migrasi
go run ./cmd/cli migrate up
go run ./cmd/cli migrate status
go run ./cmd/cli migrate down --steps 1
go run ./cmd/cli migrate down          # rollback semua

# 3. Seed (aman diulang)
go run ./cmd/cli seed

# 4. Tenant pertama
go run ./cmd/cli create-tenant `
  --name "Toko Demo" --slug toko-demo `
  --owner-name "Budi" --owner-email "owner@toko.id" --owner-password "Rahasia123" `
  --outlet-name "Outlet Utama" --outlet-code MAIN `
  --phone "08123456789" --address "Jl. Contoh No. 1"

# 5. API
go run ./cmd/api
```

Database yang dibuat `db-setup`:

- `fixphone_dev` — untuk aplikasi,
- `fixphone_test` — untuk test integrasi.

`db-drop` hanya mengizinkan nama berakhiran `_test`/`_dev` tanpa `--force`.

---

## 8. Docker Compose

Untuk mesin dengan Docker, tersedia PostgreSQL 16:

```bash
docker compose up -d
docker compose logs -f db
docker compose down
```

`docker-compose.override.yml` membuka port `127.0.0.1:55432` untuk development
(sama dengan DSN portable, sehingga `.env` tidak perlu diubah).
`deploy/docker/postgres/init.sql` menyiapkan extension dan database
`fixphone_test` pada volume kosong; skema tetap dibuat oleh migrasi aplikasi.

> Catatan: Docker tidak tersedia di mesin pengembangan saat Tahap 2, sehingga
> Compose belum dieksekusi lokal. Berkas disediakan untuk parity dan akan diuji
> pada Tahap 12 (deployment).

---

## 9. Endpoint Health & Meta

| Method | Path | Perilaku |
|---|---|---|
| GET | `/healthz` | `200` selama proses hidup; tanpa detail database |
| GET | `/readyz` | `200` bila DB up, migrasi versi 15, tidak dirty; selain itu `503` |
| GET | `/version` | nama, versi, commit, tanggal build, versi Go |

Alias `/api/v1/healthz`, `/api/v1/readyz`, `/api/v1/version` juga tersedia.

Contoh respons nyata:

```json
{"status":"ok","time":"2026-11-03T04:24:42Z","version":"0.2.0"}
```

```json
{"database":"up","dirty":false,"migration_expected":15,"migration_version":15,"status":"ready","time":"2026-11-03T04:24:42Z"}
```

Middleware global yang sudah aktif: request ID (`X-Request-ID`), structured
access log, recovery panic, CORS eksplisit, dan batas body JSON 1 MB. Endpoint
belum mengembalikan detail internal database pada `/healthz`.

---

## 10. Hasil Pengujian

Dijalankan pada Go 1.24.5 + PostgreSQL 16.15 portable:

```text
go vet ./...                              OK
go build ./...                            OK
go test ./... -count=1                    OK
  internal/config                          OK
  internal/database                        OK
  tests/integration                        OK (~19s)
go run ./cmd/cli db-setup                 OK (fixphone_dev dibuat)
go run ./cmd/cli migrate up               OK (versi 15, dirty=false)
go run ./cmd/cli migrate up (ulang)       OK (no change)
go run ./cmd/cli seed (2x + 1x)           OK (5/48/123/1, idempoten)
go run ./cmd/cli create-tenant            OK (tenant + outlet + owner)
create-tenant slug duplikat               DITOLAK sesuai harapan
GET /healthz, /readyz, /version           OK (200)
GET /api/v1/readyz                        OK (200)
route tidak dikenal                        OK (404 JSON)
```

Test integrasi mencakup:

- migrasi `up → down → up` pada database kosong,
- jumlah tabel (31), trigger `updated_at` (17), dan index kritis,
- penolakan stok negatif, status ilegal, pasangan kredensial tidak lengkap,
  tiket duplikat, HP pelanggan duplikat, invoice overpaid, dan tanggal garansi
  terbalik,
- tiket yang sama di tenant berbeda tetap diperbolehkan,
- seed dua kali tidak menggandakan role/permission/mapping/kategori,
- provisioning tenant dan penolakan slug/email duplikat serta password lemah,
- `/readyz` mengembalikan `503` sebelum migrasi dan `200` sesudahnya,
- CORS hanya mengizinkan origin terdaftar.

---

## 11. Keamanan Tahap 2

- Password dan super admin memakai bcrypt cost 12; password minimal 8 karakter,
  harus mengandung huruf dan angka, maksimal 72 byte.
- `.env` diabaikan Git; `.env.example` hanya berisi placeholder.
- Tidak ada secret yang ditulis ke log; access log hanya mencatat method, path,
  status, latency, IP, dan user agent.
- Email dinormalisasi (lowercase + trim) dan divalidasi sebelum disimpan.
- Semua operasi database memakai parameter binding; nama database divalidasi
  dengan regex sebelum `CREATE`/`DROP DATABASE`.
- CORS memakai daftar origin eksplisit, bukan wildcard.
- `audit_logs` append-only (tanpa `updated_at`).
- Seed menolak email super admin yang sudah dipakai pengguna tenant, sehingga
  tidak bisa terjadi eskalasi role lewat tabrakan email.
- Kredensial perangkat disimpan sebagai `bytea` terenkripsi (kolom sudah ada);
  implementasi enkripsi/dekripsi dan permission reveal masuk Tahap 6.

---

## 12. Yang Sengaja Ditunda

| Item | Tahap |
|---|---|
| RLS PostgreSQL + `SET LOCAL app.tenant_id` | 11 |
| Autentikasi JWT, refresh rotation, RBAC middleware | 3 |
| Enkripsi/dekripsi PIN perangkat & kredensial WA | 6/9 |
| Worker reminder & WhatsApp outbox | 9 |
| Backup/restore otomatis + restore drill | 11–12 |
| Docker image API, Nginx, CI/CD | 12 |
| Frontend React (login, shell, modul) | 3+ |

---

## 13. Definition of Done Tahap 2

- [x] `migrate up` membuat seluruh 31 tabel tanpa error.
- [x] `migrate down` bersih pada database kosong, lalu `up` kembali berhasil.
- [x] Constraint kritis menolak data invalid.
- [x] `seed` dua kali tidak menggandakan data.
- [x] `/healthz` dan `/readyz` merespons benar (termasuk `503` saat belum siap).
- [x] `go vet ./...`, `go build ./...`, dan `go test ./...` lulus.
- [x] `.env.example`, portable DB, Docker Compose, Makefile, dan dokumentasi ada.
