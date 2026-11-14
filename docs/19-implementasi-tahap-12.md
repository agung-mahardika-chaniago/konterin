# 19 — Implementasi Tahap 12: Deployment Production

**Status:** selesai (validasi container dijalankan CI) · **Versi aplikasi:** 1.0.0 · **Tanggal:** 13 November 2026
**Acuan:** [`05 — Roadmap MVP`](05-roadmap-mvp.md) Tahap 12, [`07 — Keamanan`](07-keamanan.md), [`20 — Runbook Operasional`](20-runbook-operasional.md)

---

## 1. Tujuan Tahap 12

Tahap 12 membawa FixPhone dari aplikasi yang lolos uji menjadi layanan yang
bisa dioperasikan: satu perintah Compose menyalakan database, migrasi, seed,
API, worker, SPA, dan TLS; backup berjalan terjadwal; kegagalan terlihat lewat
metrik dan alert; serta ada prosedur rollback dan restore yang teruji.

Cakupan yang diselesaikan:

1. **Runtime RLS penuh** — request tenant berjalan sebagai role
   `fixphone_app` di dalam satu transaksi, sedangkan login/refresh/tracking
   publik/migrasi memakai pool admin.
2. **Image production** multi-stage, non-root, health check, read-only root
   filesystem untuk API/worker.
3. **Compose production** dengan Caddy (TLS otomatis), Nginx unprivileged
   (SPA + reverse proxy), PostgreSQL internal, worker, backup, dan monitoring
   opsional.
4. **Bootstrap sekali jalan**: `migrate up` → `seed` → `set-app-role` → `doctor`
   sebelum API dinyalakan.
5. **Backup harian + offsite + retensi + drill restore mingguan** dan metrik
   backup untuk alert.
6. **Monitoring dasar**: uptime API/publik, error rate, latensi, disk, backup
   basi/gagal, sertifikat TLS.
7. **CI** membangun ketiga image, memindai CVE CRITICAL dengan Trivy, dan
   menjalankan stack production + smoke test + ZAP baseline (advisory).
8. **Seed production tanpa demo data** dan verifikasi `doctor`.
9. **Smoke test** PowerShell/Bash dan dokumentasi handover.

---

## 2. Arsitektur Deployment

```
                       Internet
                          │ 80/443
                    ┌─────▼─────┐
                    │   Caddy   │  TLS otomatis (ACME), HSTS, gzip/zstd
                    │   edge    │  redirect HTTP → HTTPS
                    └─────┬─────┘
                          │ http (fixphone-edge)
                    ┌─────▼──────────┐
                    │ Nginx (uid101) │  SPA + /api & /uploads reverse proxy
                    │     web        │  security header, cache aset
                    └─────┬──────────┘
                          │ http (fixphone-internal)
              ┌───────────┴────────────┐
        ┌─────▼─────┐            ┌─────▼─────┐
        │  api      │            │  worker   │
        │ RLS role  │            │ admin DSN │
        └─────┬─────┘            └─────┬─────┘
              │                        │
        ┌─────▼────────────────────────▼─────┐
        │        PostgreSQL 16 (pgdata)      │
        │  fixphone_app (RLS) + owner        │
        └─────▲──────────────────────────────┘
              │
        ┌─────┴──────┐        ┌───────────────────┐
        │  backup    │        │ Prometheus/Alert  │ (profile monitoring)
        │ cron+drill │        │ node/blackbox     │
        └────────────┘        └───────────────────┘
```

Hanya Caddy yang membuka port host. PostgreSQL, API, worker, dan exporter
berada di jaringan `fixphone-internal`; Nginx satu-satunya service yang
menyentuh kedua jaringan.

---

## 3. Runtime RLS (perubahan terpenting)

### 3.1 Dua pool koneksi

| Pool | Env | Dipakai untuk |
|---|---|---|
| Aplikasi | `DB_DSN` (`fixphone_app`, NOSUPERUSER, NOBYPASSRLS) | Semua request setelah tenant diketahui |
| Admin | `DB_ADMIN_DSN` (pemilik skema) | Login, refresh token, resolusi tenant, tracking publik, worker, migrasi/seed, backup manual |

`config.Load()` mewajibkan `DB_ADMIN_DSN` di production. Bila `BACKUP_DSN`
kosong, nilainya otomatis memakai `DB_ADMIN_DSN` agar tombol backup manual per
tenant tetap bekerja tanpa menduplikasi secret.

### 3.2 Transaksi request

`middleware.Tenant` melakukan, untuk setiap request tenant:

```go
tx := appDB.WithContext(ctx).Begin()
tx.Exec("SELECT set_config('app.tenant_id', ?, true)", tenantID)
tx.Exec("SET LOCAL statement_timeout = '15s'")
ctx = database.ContextWithTx(ctx, tx)
c.Request = c.Request.WithContext(ctx)
```

- `SET LOCAL` mengikat tenant hanya pada transaksi tersebut; koneksi yang
  kembali ke pool tidak membawa `app.tenant_id`.
- Response `2xx/3xx` → commit; response `4xx/5xx`, panic, atau gagal commit →
  rollback. Tidak ada penulisan setengah jadi.
- Superadmin platform (`tenant_id IS NULL`) tidak membuka transaksi tenant.

### 3.3 Binding GORM

Repository menyimpan handle `*gorm.DB` berumur panjang dari saat router dibuat.
Callback `database.EnableTenantBinding(appDB)` menulis ulang `ConnPool` setiap
statement menjadi transaksi request bila context membawanya. `SkipDefaultTransaction: true`
mencegah GORM membuat transaksi implisit yang commit lebih awal.

Transaksi bersarang **wajib** memakai `database.WithContext(ctx, db)` agar GORM
membuat savepoint pada transaksi request, bukan koneksi baru tanpa tenant:

```go
return database.WithContext(ctx, s.db).Transaction(func(tx *gorm.DB) error { ... })
```

Seluruh service telah dimigrasikan ke pola ini (auth, servis, inventori,
invoice, pembayaran, pengeluaran, garansi, notifikasi, pengaturan, pengguna,
laporan).

### 3.4 Bukti uji

`backend/tests/integration/rls_gorm_test.go` menambahkan tiga test yang memakai
jalur produksi (GORM + binding + request transaction):

1. Akses tanpa context tenant = 0 baris; dengan context = hanya tenant aktif.
2. Insert lintas tenant ditolak `row-level security`; transaksi di-rollback dan
   tidak ada baris yang bertahan.
3. Transaksi bersarang memakai savepoint: rollback savepoint tidak membatalkan
   pekerjaan luar, sedangkan savepoint sukses ikut commit bersama request.

Test lama `rls_test.go` (SQL mentah) tetap berjalan sebagai lapisan kedua.

---

## 4. Image & Hardening Container

### 4.1 Backend

`deploy/docker/backend.Dockerfile`:

- Build multi-stage `golang:1.26.11-alpine`, `CGO_ENABLED=0`, `-trimpath`,
  ldflags mengisi versi/commit/tanggal.
- Satu image berisi `fixphone-api`, `fixphone-worker`, `fixphone-cli`.
- Runtime `alpine:3.21`, user **10001:10001**, hanya `ca-certificates`, `tzdata`,
  `wget` (health check).
- `/app/uploads` dan `/app/backups` dimiliki user aplikasi.
- `HEALTHCHECK` memakai `/healthz`; Compose memakai `/readyz`.

### 4.2 Frontend

`deploy/docker/frontend.Dockerfile`:

- Build `node:22-alpine` → `npm ci` → `npm run build`.
- Runtime `nginxinc/nginx-unprivileged:1.30-alpine`, user **101**, listen 8080.
- `deploy/nginx/nginx.conf` menangani SPA fallback, cache aset hash 1 tahun,
  gzip, security header, dan proxy `/api` + `/uploads`. `/readyz` dan
  `/version` diproxyeksplisit ke API agar fallback SPA tidak menjawab `200`
  dengan HTML; `/metrics` sengaja `404` di edge.

### 4.3 Backup

`deploy/backup/Dockerfile` berisi `postgresql-client`, `age`, `rclone`, `tini`.
Container menjalankan cron internal dan menulis metrik textfile ke volume
terpisah yang dibaca `node-exporter`.

### 4.4 Kontrol runtime Compose

- `api` dan `worker`: `read_only: true`, `tmpfs /tmp`, `cap_drop: ALL`,
  `no-new-privileges`, `init: true`.
- Database tanpa port host; hanya Caddy yang publish 80/443.
- Log dibatasi `20m × 5` per container.
- `stop_grace_period: 30s` untuk API agar request berjalan selesai.

---

## 5. Compose Production

File: `docker-compose.prod.yml` + `.env.production.example`.

`APP_ENV=production` mengaktifkan validasi: `APP_BASE_URL` wajib HTTPS,
`CORS_ORIGINS` wajib diisi, `SEED_DEMO=false`, dan `DB_ADMIN_DSN` wajib ada.
Nilai rahasia memakai interpolasi `:?` sehingga Compose menolak start bila
belum diisi.

Service:

| Service | Peran |
|---|---|
| `db` | PostgreSQL 16, checksum, healthcheck, volume `pgdata` |
| `migrate` | Sekali jalan: migrasi, seed, aktifkan login role RLS, doctor |
| `api` | API Gin dengan pool RLS + admin |
| `worker` | Outbox WA, reminder, purge kredensial (admin DSN) |
| `web` | Nginx unprivileged, SPA + proxy API |
| `caddy` | TLS otomatis, HSTS, redirect HTTP, edge |
| `backup` | Cron dump/offsite/drill + metrik |
| `prometheus`, `alertmanager`, `node-exporter`, `blackbox-exporter` | Profile `monitoring` |

Perintah:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build
```

---

## 6. Bootstrap, Migrasi & Rollback

### 6.1 Urutan bootstrap

Service `migrate` menjalankan:

```sh
fixphone-cli migrate up
fixphone-cli seed
fixphone-cli set-app-role
fixphone-cli doctor
```

- `seed` idempoten dan **tidak** membuat data demo (`SEED_DEMO=false`); password
  superadmin yang sudah ada tidak direset.
- `set-app-role` mengaktifkan `LOGIN` dan mengganti password role aplikasi
  memakai `format(%L)` (aman terhadap quoting), hanya lewat `DB_ADMIN_DSN`.
- `doctor` memeriksa versi migrasi, status dirty, 30 tabel/policy RLS, atribut
  role (login, bukan superuser, bukan bypassrls), dan sisa tenant demo. Exit
  non-zero menghentikan `api`/`worker` karena `depends_on:
  service_completed_successfully`.

### 6.2 Migrasi

- `AUTO_MIGRATE` default `false`; production memakai service sekali jalan.
- Migrasi berjalan dengan admin DSN. Role aplikasi tidak boleh memiliki DDL.
- Status dibaca `/readyz`; API menolak ready bila versi tidak sama atau dirty.

### 6.3 Rollback

Rollback aplikasi dan database dipisahkan:

1. **Aplikasi**: deploy image versi sebelumnya dengan tag yang sama
   (`FIXPHONE_VERSION`), tanpa menjalankan migrasi down.
2. **Database**: hanya bila migrasi terakhir kompatibel ke belakang, jalankan
   `migrate down --steps 1` dari image versi sebelumnya; jika tidak, restore
   dump pra-rilis ke database baru lalu promosikan.
3. **Darurat**: `migrate force --version N` hanya untuk memperbaiki status dirty
   setelah backup, bukan untuk melewati migrasi.

Prosedur lengkap ada di [`20 — Runbook Operasional`](20-runbook-operasional.md) §5.

---

## 7. Backup, Restore & Drill

`deploy/backup/backup.sh` (harian):

1. `pg_dump --format=custom --compress=9 --no-owner --no-privileges`.
2. Verifikasi dump dengan `pg_restore --list`.
3. Arsip `uploads` (read-only volume) menjadi `tar.gz`.
4. Opsional enkripsi `age` (`BACKUP_AGE_RECIPIENT`).
5. Opsional salin offsite `rclone` (`BACKUP_REMOTE`), retensi offsite.
6. Hapus backup lokal lebih lama dari `BACKUP_RETENTION_DAYS` (default 14).
7. Tulis `backup.prom` (`last_success_timestamp`, `last_status`, ukuran,
   durasi) ke volume `backupstatus`.

`deploy/backup/drill.sh` (mingguan, aktif bila `RESTORE_DRILL_DSN` diisi):

1. Ambil dump terbaru (dekripsi `age` bila perlu).
2. `pg_restore --clean --if-exists --single-transaction` ke database scratch.
3. Hitung tabel `public`; gagal bila < `RESTORE_DRILL_MIN_TABLES` (25).
4. Tulis metrik drill.

`deploy/backup/restore.sh` untuk restore manual dengan konfirmasi `--yes`.
Runbook menekankan restore ke database baru lalu verifikasi sebelum promosi.

---

## 8. Monitoring & Alert

API mengekspos `/metrics` tanpa dependency tambahan:

- `fixphone_build_info`, `fixphone_uptime_seconds`, `fixphone_db_up`
- `fixphone_http_requests_total{method,route,status}`
- `fixphone_http_request_duration_seconds` (histogram)

`METRICS_TOKEN` opsional melindungi endpoint dengan Bearer token; Nginx
mengembalikan 404 untuk `/metrics` dan Prometheus men-scrape `api:8080` di
jaringan internal.

Alert (`deploy/monitoring/alerts.yml`):

| Alert | Ambang |
|---|---|
| `FixPhoneApiDown` | scrape gagal 2 menit |
| `FixPhonePublicEndpointDown` | probe HTTPS gagal 3 menit |
| `FixPhoneDatabaseDown` | `fixphone_db_up == 0` 2 menit |
| `FixPhoneTLSCertExpiring` | sertifikat < 14 hari |
| `FixPhoneApiErrorRateHigh` | 5xx > 5% selama 5 menit |
| `FixPhoneLatencyHigh` | p95 > 2 detik selama 10 menit |
| `FixPhoneDiskSpaceLow` | disk < 15% |
| `FixPhoneDiskWillFill` | prediksi penuh < 24 jam |
| `FixPhoneBackupFailed` / `Stale` | status 0 / sukses terakhir > 26 jam |
| `FixPhoneRestoreDrillStale` | drill sukses terakhir > 8 hari |

Monitoring bersifat opt-in:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml \
  --profile monitoring up -d
```

---

## 9. CI: Build, Scan & Smoke

`.github/workflows/ci.yml` sekarang berisi tujuh job:

| Job | Isi |
|---|---|
| `backend` | build, vet, seluruh test (termasuk RLS GORM + integrasi) |
| `frontend` | typecheck, Vitest, build |
| `security` | gosec, govulncheck, npm audit |
| `restore-drill` | dump + restore + verifikasi |
| `images` | build backend/frontend/backup, Trivy CRITICAL (`ignore-unfixed`) |
| `deploy-smoke` | Compose production + tenant + smoke + ZAP advisory |
| `e2e` | Playwright alur servis |

`deploy-smoke` memakai `APP_DOMAIN=localhost` sehingga Caddy memakai CA
internal, lalu menjalankan `scripts/smoke-prod.sh` dengan `SMOKE_INSECURE=1`.
Ini menguji jalur nyata: image, migrasi, seed, RLS, TLS, proxy, cookie, dan
API. ZAP baseline dijalankan terhadap `http://web:8080` di jaringan Compose dan
bersifat advisory (laporan diunggah sebagai artefak).

---

## 10. Smoke Test Production

Dua skrip setara:

```powershell
$env:SMOKE_BASE_URL="https://fixphone.example.com"
$env:SMOKE_EMAIL="owner@toko.id"
$env:SMOKE_PASSWORD="..."
pwsh -NoProfile -File scripts/smoke-prod.ps1
```

```bash
SMOKE_BASE_URL=https://fixphone.example.com \
SMOKE_EMAIL=owner@toko.id SMOKE_PASSWORD='...' \
./scripts/smoke-prod.sh
```

Pemeriksaan: `/healthz`, `/readyz` (JSON `ready` + database `up`), `/version`
(metadata build), HSTS (bila HTTPS), penolakan anonim, tracking palsu `404`,
login, profil, dashboard, daftar servis & pelanggan, rotasi refresh cookie,
logout, dan refresh setelah logout harus `401`. Exit code non-zero bila ada
kegagalan.

---

## 11. Seed Production

- `SEED_DEMO=false` diwajibkan validasi production.
- Seed hanya membuat role, permission, superadmin awal, dan data sistem.
- Superadmin yang sudah ada tidak diubah; password tidak direset.
- Login pertama memaksa ganti password (`must_change_password`).
- `doctor` memperingatkan tenant dengan nama/slug mengandung `demo`.
- Tenant pertama dibuat eksplisit dengan `fixphone-cli create-tenant` (owner +
  outlet + pengaturan), bukan data contoh.

---

## 12. Verifikasi & Hasil

Dijalankan di mesin pengembangan (Docker tidak tersedia):

| Pemeriksaan | Hasil |
|---|---|
| `gofmt` | bersih |
| `go build ./...` | lulus |
| `go vet ./...` | lulus |
| `go test ./... -count=1` | lulus (integrasi ±228 dtk, termasuk 3 test RLS GORM baru) |
| `npm run typecheck` | lulus |
| `npm test` | 14 test lulus |
| `npm run build` | lulus |
| YAML Compose/alert/template | valid (parser YAML) |
| Script shell backup/entrypoint | ditinjau statis; dijalankan di image CI |

Keterbatasan: build image, `docker compose up`, TLS, dan ZAP **tidak dapat
dijalankan lokal** karena Docker tidak terpasang. Langkah tersebut dijalankan
otomatis oleh job `images` dan `deploy-smoke` di CI. Acceptance Tahap 12
diverifikasi ulang pada CI run pertama setelah push; bila ada kegagalan
lingkungan, perbaikan dilakukan di workflow, bukan diabaikan.

---

## 13. Risiko & Tindak Lanjut

| Risiko | Mitigasi |
|---|---|
| Container belum diuji lokal | CI `images` + `deploy-smoke`; jalankan smoke manual setelah deploy pertama |
| Caddy butuh port 80/443 & DNS benar | Checklist go-live; staging dengan domain uji |
| Offsite backup butuh kredensial rclone | Uji `rclone lsd` sebelum go-live; alert backup basi |
| Metrik process-local | Satu instance API; multi-instance perlu scrape per replika/shared store |
| Restore drill butuh DB scratch | Sediakan `fixphone_drill`; alert drill basi bila tidak |
| Rotasi `APP_ENC_KEY` belum otomatis | Prosedur manual di runbook; versioning kunci sudah ada di skema |

---

## 14. Acceptance Tahap 12

- [x] Compose production lengkap (Caddy TLS, Nginx, API, worker, DB, backup).
- [x] Image multi-stage, non-root, health check, read-only untuk API/worker.
- [x] Migrasi + seed + role RLS + doctor otomatis sebelum API.
- [x] Rollback plan terdokumentasi.
- [x] Backup harian, retensi, offsite opsional, drill mingguan, restore manual.
- [x] Monitoring uptime/error/latency/disk/backup/TLS + alert.
- [x] CI build image, Trivy, smoke production, ZAP advisory.
- [x] Seed production tanpa demo data.
- [x] Smoke test PowerShell/Bash + go-live checklist.
- [x] Runbook operasional dan panduan admin/teknisi.
- [x] Versi aplikasi `1.0.0`.
