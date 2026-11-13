# 18 — Implementasi Tahap 11: Testing & Security Review

**Status:** selesai · **Versi aplikasi:** 0.11.0 · **Tanggal:** 12 November 2026
**Acuan:** [`05 — Roadmap MVP`](05-roadmap-mvp.md) Tahap 11, [`07 — Keamanan`](07-keamanan.md), [`06 — Kontrak API`](06-kontrak-api.md)

---

## 1. Tujuan Tahap 11

Tahap 11 memastikan aplikasi aman dan stabil sebelum masuk Tahap 12
(deployment production). Fokusnya bukan fitur baru, melainkan bukti bahwa
kontrol keamanan benar-benar bekerja dan alur bisnis utama tidak regresi.

Cakupan yang diselesaikan:

1. **RLS PostgreSQL** untuk seluruh tabel ber-`tenant_id`, diuji dengan role
   aplikasi non-superuser.
2. **Test IDOR lintas tenant** untuk daftar, baca, tulis, dan data turunan.
3. **Uji konkurensi** nomor tiket, stok sparepart, dan pembayaran.
4. **Header keamanan** API-wide (CSP, Permissions-Policy, Referrer-Policy,
   HSTS di production).
5. **Restore drill** penuh `pg_dump` → `pg_restore` dengan verifikasi data,
   policy RLS, dan versi migrasi.
6. **Unit test frontend (Vitest)** dan **E2E Playwright** alur servis lengkap
   + audit aksesibilitas kritis.
7. **Uji beban ringan k6**: 50 VU, target p95 < 500 ms.
8. **Security scan**: `gosec`, `govulncheck`, `npm audit`, pembaruan dependency
   rentan.
9. **CI GitHub Actions** yang menjalankan seluruh gate di atas.

---

## 2. File yang Ditambahkan / Diubah

### Backend

```
backend/migrations/000019_rls.up.sql              # role aplikasi, grant, ENABLE RLS, 30 policy
backend/migrations/000019_rls.down.sql            # lepas policy/grant tanpa menghapus role
backend/internal/database/migrate.go              # ExpectedVersion = 19
backend/internal/middleware/security.go           # SecurityHeaders(enableHSTS)
backend/internal/middleware/security_test.go      # test header + origin
backend/internal/router/router.go                 # pasang header API-wide
backend/internal/modules/payment/service.go       # 404 untuk invoice tenant lain
backend/internal/modules/inventory/stock.go       # 404 untuk sparepart tenant lain
backend/tests/integration/rls_test.go             # 5 test RLS dengan role non-superuser
backend/tests/integration/tenant_isolation_test.go# matriks IDOR lintas tenant
backend/tests/integration/concurrency_test.go     # stok & pembayaran paralel
backend/go.mod                                    # toolchain go1.26.11, pgx v5.9.2
```

### Frontend

```
frontend/vitest.config.ts                         # unit test jsdom
frontend/playwright.config.ts                     # E2E terhadap stack live
frontend/src/features/invoices/format.test.ts     # 6 test formatter invoice
frontend/src/features/dashboard/format.test.ts    # 8 test formatter dashboard
frontend/e2e/service-flow.spec.ts                 # alur servis penuh + axe smoke
frontend/package.json                             # script test, test:watch, e2e; bump deps
```

### Infrastruktur

```
scripts/restore-drill.ps1                         # backup + restore + verifikasi
scripts/load/k6-smoke.js                          # 50 VU read-only smoke
.github/workflows/ci.yml                          # backend, frontend, security, restore, E2E
```

---

## 3. RLS & Isolasi Tenant

### 3.1 Desain migrasi `000019_rls`

Migrasi membuat role aplikasi `fixphone_app` secara idempoten:

```sql
CREATE ROLE fixphone_app NOLOGIN NOSUPERUSER NOBYPASSRLS NOCREATEDB NOCREATEROLE;
```

- Role bersifat **cluster-wide**, sehingga migrasi `down` **tidak** menghapusnya.
  Password di production diatur DBA di luar repo (`ALTER ROLE ... PASSWORD ...`).
- Aplikasi produksi nantinya terhubung memakai role ini. Test membuat role bisa
  `LOGIN` hanya pada database uji yang dapat dibuang.

Policy memakai GUC `app.tenant_id` dan **fail-closed** bila tidak diisi:

```sql
USING (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
WITH CHECK (tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid)
```

Cakupan:

| Kelompok | Jumlah | Perlakuan |
|---|---:|---|
| Tabel langsung dengan `tenant_id` | 26 | policy `tenant_isolation` |
| `tenants` | 1 | policy berdasarkan `id` |
| Tabel turunan (`refresh_tokens`, `user_outlets`, `notification_reads`) | 3 | policy lewat join ke tabel induk |
| `roles`, `permissions`, `role_permissions` | 3 | dikecualikan (referensi global) |

Total **30 tabel ber-RLS dan 30 policy**. `audit_logs` diberi policy tenant dan
tambahan `REVOKE UPDATE, DELETE` dari role aplikasi agar append-only
(docs/07 §10).

### 3.2 Test isolasi database

`backend/tests/integration/rls_test.go` terhubung sebagai `fixphone_app` (bukan
superuser, tanpa `BYPASSRLS`) dan membuktikan:

1. `TestRLSRequiresTenantContext` — GUC kosong/tidak di-set menghasilkan nol
   baris dan INSERT ditolak (`fail-closed`).
2. `TestRLSScopesRowsToActiveTenant` — tenant A hanya melihat barisnya sendiri.
3. `TestRLSBlocksCrossTenantWrites` — INSERT lintas tenant ditolak; UPDATE dan
   DELETE lintas tenant tidak mengenai baris apa pun.
4. `TestRLSProtectsIndirectTenantTables` — tabel turunan ikut terisolasi.
5. `TestRLSAuditLogIsAppendOnly` — role aplikasi tidak dapat `UPDATE`/`DELETE`
   audit log.

Superuser mem-bypass RLS, sehingga pembuktian wajib memakai role aplikasi.
**Wiring runtime** (`SET LOCAL app.tenant_id` per request) belum dipasang di
Tahap 11 dan dijadwalkan pada Tahap 12; lihat §11.

### 3.3 Matriks IDOR aplikasi

`TestTenantIDORMatrix` menjalankan dua tenant nyata dan mencoba:

- daftar dengan filter/ID milik tenant lain,
- baca detail lewat ID tebakan (customer, service order, diagnosis, item,
  timeline, invoice, payment, sparepart, user, warranty, claim, expense,
  backup, kredensial perangkat),
- tulis/ubah/hapus lintas tenant,
- memastikan data tenant korban tetap utuh setelah percobaan.

Dua temuan diperbaiki:

1. `payment.Service.ensureInvoiceAccess` — `GET /invoices/:id/payments`
   sebelumnya membocorkan ada/tidaknya invoice tenant lain; kini **404**.
2. `inventory.ListMovements` — filter `sparepart_id` eksplisit kini
   memverifikasi sparepart berada di tenant aktif; jika tidak, **404**.

---

## 4. Uji Konkurensi

`backend/tests/integration/concurrency_test.go`:

- **Stok tidak pernah oversell**: 8 permintaan paralel terhadap stok 3
  menghasilkan tepat 3 sukses dan 5 `INSUFFICIENT_STOCK`, dengan 3 pergerakan
  `service_out`.
- **Pembayaran idempoten & tidak overpay**: retry dengan idempotency key yang
  sama hanya menghasilkan satu pembayaran; total melebihi sisa tagihan ditolak
  `OVERPAYMENT`. Hasil akhir: 3 pembayaran `posted`, `paid_amount = 180000`.

Nomor tiket sudah diuji pada integration test Tahap 5; tidak ada perubahan
skema penomoran pada Tahap 11.

---

## 5. Header Keamanan

`SecurityHeaders(enableHSTS bool)` dipasang untuk seluruh response API:

| Header | Nilai |
|---|---|
| `Content-Security-Policy` | `default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'` |
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `Referrer-Policy` | `strict-origin-when-cross-origin` |
| `Permissions-Policy` | kamera, mikrofon, geolokasi, dan fitur sensitif dimatikan |
| `Strict-Transport-Security` | hanya saat `APP_ENV=production` |
| `Cache-Control` | `no-store` untuk response API |

HSTS tidak dikirim di development agar browser lokal tidak mengunci
`http://localhost`. Perilaku ini diuji di `security_test.go`.

---

## 6. Backup & Restore Drill

`scripts/restore-drill.ps1` membuktikan backup penuh dapat dipulihkan:

1. `pg_dump -Fc --no-owner --no-privileges` dari database sumber.
2. Buat database scratch baru (`TEMPLATE template0`).
3. Pastikan role `fixphone_app` ada (policy mereferensikannya).
4. `pg_restore --no-owner --exit-on-error -j 2`.
5. Bandingkan jumlah tabel dan jumlah baris **setiap** tabel.
6. Bandingkan versi `schema_migrations`, jumlah policy, dan jumlah tabel RLS.
7. Hapus database scratch dan file dump (kecuali `-KeepDump`/`-KeepScratch`).

Hasil lokal (PostgreSQL 16.15, database uji Tahap 11):

```
Restore drill LULUS
  dump     : 162338 byte
  tabel    : 34
  policy   : 30
  migrasi  : 19
```

Script mencari `pg_dump`/`pg_restore`/`psql` di `PATH`, instalasi PostgreSQL
standar, `.dev/pgsql/bin`, lalu `.dev/tools/pgsql-client`. Job CI yang sama
berjalan di Linux memakai `pwsh` dan `postgresql-client-16`.

Backup tenant dari UI (`GET /settings/backups/:id/download`) adalah dump
per-tenant untuk keperluan ekspor/manual, **bukan** artefak restore independen;
restore resmi memakai dump penuh database (docs/07 §14).

---

## 7. Test Frontend

### 7.1 Unit test (Vitest)

- `frontend/src/features/invoices/format.test.ts` — 6 test: placeholder tanggal,
  format `id-ID`, label status/metode/item, tanggal hari ini.
- `frontend/src/features/dashboard/format.test.ts` — 8 test: rupiah, unit
  kompak, angka, durasi, sapaan, inisial, kunci tanggal lokal, rentang inklusif.

Hasil: **14/14 lulus**.

### 7.2 E2E Playwright

`frontend/e2e/service-flow.spec.ts` menjalankan dua skenario:

1. **Alur servis lengkap** — login → intake pelanggan/perangkat → diagnosis →
   tambah item jasa → finalisasi invoice → pembayaran 200000 → transisi status
   lewat API (`waiting_diagnosis → in_progress → testing → completed →
   ready_for_pickup → picked_up`) → verifikasi status "Sudah Diambil" →
   tracking publik dengan nomor tiket + 4 digit terakhir HP.
2. **Axe smoke** halaman `/login` — tidak boleh ada pelanggaran aksesibilitas
   berdampak `critical` (WCAG 2.0 A/AA).

Transisi status dilakukan lewat sesi API terpisah agar test tidak bergantung
pada rantai dialog yang rapuh; state machine tetap ditegakkan server dan diuji
di Go. Hasil lokal: **2/2 lulus**.

Perintah:

```powershell
# terminal 1
go run ./cmd/api

# terminal 2
npm run dev

# terminal 3
npx playwright install chromium
npm run e2e
```

Environment default mengikuti tenant development terdokumentasi:
`E2E_EMAIL=owner@fixphone.local`, `E2E_PASSWORD=Owner#12345`,
`E2E_BASE_URL=http://localhost:5173`.

---

## 8. Uji Beban k6

`scripts/load/k6-smoke.js` mensimulasikan 50 VU selama ±105 detik (ramp-up 30 s,
steady 60 s, ramp-down 15 s) terhadap endpoint baca:

- `/dashboard/summary`, `/dashboard/charts?range=7`
- `/service-orders`, `/invoices`, `/customers`, `/spareparts`

Threshold: `http_req_failed < 1%`, `http_req_duration p95 < 500 ms`.

Hasil lokal:

```
checks_succeeded...: 100.00% 5485/5485
http_req_failed....: 0.00%   0/2743
http_req_duration..: avg=8.13ms p(95)=14.47ms max=561.66ms
vus_max............: 50
```

Menjalankan ulang:

```powershell
k6 run -e BASE_URL=http://127.0.0.1:8080/api/v1 `
       -e EMAIL=owner@fixphone.local -e PASSWORD='Owner#12345' `
       scripts/load/k6-smoke.js
```

k6 bukan dependency aplikasi; pasang dari <https://k6.io/docs/get-started/installation/>
atau letakkan binary di `.dev/tools/k6` (di luar Git).

---

## 9. Dependency & Security Scan

| Alat | Cakupan | Hasil |
|---|---|---|
| `gosec -quiet ./...` | backend | **0 temuan** |
| `govulncheck ./...` | backend | **0 vulnerability yang dipanggil** |
| `npm audit --omit=dev --audit-level=high` | dependency produksi frontend | **exit 0** (hanya 2 moderate react-router) |
| `npm audit` (termasuk dev) | frontend | 8 temuan dev-only, didokumentasikan |

Perbaikan yang dilakukan:

- **Go toolchain** `go1.26.0` → `go1.26.11` melalui `toolchain` di `go.mod`.
  Versi stdlib lama memiliki 35 advisory yang terpanggil.
- **pgx** `v5.7.2` → `v5.9.2` untuk `GO-2026-5004` (SQL injection melalui
  placeholder confusion).
- **react-router-dom** `6.30.1` → `^6.30.6`.
- **vite** `5.4.11` → `5.4.21`, **postcss** `8.4.49` → `8.5.32`.
- **@playwright/test** terpasang `1.65.0`.

Risiko diterima (dicatat, bukan blocker rilis internal):

- `react-router` v6 menyisakan 2 advisory **moderate** (open redirect lewat
  backslash, dan isu SSR hydration). Aplikasi tidak memakai SSR dan seluruh
  navigasi berasal dari route internal; upgrade ke v7 adalah perubahan besar
  yang dijadwalkan terpisah.
- `vitest`/`tinypool` dev-only memiliki advisory critical (prototype pollution
  di worker). Hanya aktif saat menjalankan test lokal/CI, tidak dikirim ke
  production; upgrade ke vitest 5 memerlukan migrasi test runner.
- esbuild/vite dev-server advisory sudah hilang setelah bump vite 5.4.21.

---

## 10. CI Pipeline

`.github/workflows/ci.yml` memiliki lima job:

| Job | Isi |
|---|---|
| `backend` | PostgreSQL 16 service, `go build`, `go vet`, `go test ./... -count=1` |
| `frontend` | `npm ci`, `typecheck`, `npm test`, `npm run build` |
| `security` | `gosec`, `govulncheck`, `npm audit --omit=dev --audit-level=high` |
| `restore-drill` | migrasi + seed + tenant, lalu `pwsh scripts/restore-drill.ps1` |
| `e2e` | DB + API + Vite, `playwright install --with-deps chromium`, `npm run e2e` |

Catatan:

- Semua job memakai PostgreSQL 16 sebagai service di port `55432` agar DSN sama
  dengan development.
- E2E memakai kredensial CI khusus; tidak ada secret production.
- Artefak Playwright diunggah saat gagal.
- OWASP ZAP baseline belum dijalankan otomatis di CI karena butuh target yang
  stabil dan waktu tambahan; dijadwalkan sebagai job opsional di Tahap 12
  bersama smoke test production.

---

## 11. Checklist Keamanan (docs/07 §17)

Status per Tahap 11:

- [x] Semua endpoint memiliki permission & validasi — diuji RBAC + IDOR.
- [x] Test IDOR lintas tenant lulus — `TestTenantIDORMatrix`.
- [x] RLS aktif dan diuji — 30 tabel, 30 policy, role non-superuser.
- [x] Tidak ada secret di repo/image/log — `.env` diabaikan; scan manual + gosec.
- [ ] Password default diganti — prosedur Tahap 12 (production).
- [ ] HTTPS + header keamanan terverifikasi — header sudah ada & diuji; TLS
      production Tahap 12.
- [x] Rate limit login & tracking terbukti — test Tahap 3/9 tetap lulus.
- [x] PIN tidak muncul di response mana pun — `RevealCredential` ber-audit,
      test kredensial lulus.
- [x] Audit log mencatat aksi sensitif — append-only untuk role aplikasi.
- [ ] Backup otomatis berjalan & restore teruji — restore drill lulus; jadwal
      otomatis Tahap 12.
- [x] Dependency scan bersih dari critical/high — production frontend exit 0,
      backend govulncheck 0.
- [x] `gosec` & `npm audit` ditinjau — hasil di §9.
- [x] Log tidak memuat data sensitif — review handler/logging Tahap 10.
- [ ] Runbook insiden & kontak tersedia — Tahap 12.
- [ ] Hak akses server (SSH, DB) minimum — Tahap 12.
- [ ] Tidak ada akun/demo data di production — Tahap 12.

Mapping OWASP Top 10 di docs/07 §15 tetap berlaku; tidak ada risiko baru yang
memerlukan perubahan kontrol.

---

## 12. Batasan & Langkah Berikutnya

Batasan yang disengaja pada Tahap 11:

1. **RLS belum di-wire ke runtime request.** Migrasi dan policy sudah aktif
   untuk role aplikasi, tetapi API masih terhubung sebagai role yang lebih
   tinggi. Tahap 12 akan menambahkan `SET LOCAL app.tenant_id` di transaksi per
   request (middleware), mengganti DSN ke `fixphone_app`, lalu menjalankan
   seluruh integration test kembali.
2. **E2E berjalan pada satu browser (Chromium)**; matrix Firefox/WebKit belum
   ditambahkan.
3. **Audit aksesibilitas** terbatas pada axe `critical` di halaman login;
   audit keyboard/kontras menyeluruh dan halaman lain dijadwalkan bersama
   perbaikan UI berikutnya.
4. **ZAP baseline** belum otomatis; lihat §10.
5. **`npm audit` dev-only** masih memiliki temuan critical pada `tinypool`
   (vitest). Tidak memengaruhi artefak production.

Langkah berikutnya: **Tahap 12 — Deployment Production** (Docker multi-stage,
TLS/header production, migrasi otomatis, backup terjadwal + offsite, monitoring,
smoke test production, runbook & handover).
