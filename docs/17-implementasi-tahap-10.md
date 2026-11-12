# 17 — Implementasi Tahap 10: Laporan, Pengaturan & Dashboard Bisnis

**Status:** selesai ✅ **Versi aplikasi:** 0.10.0 ✅ **Tanggal:** 11 November 2026
**Acuan:** [`05 — Roadmap MVP`](05-roadmap-mvp.md) Tahap 10, [`06 — Kontrak API`](06-kontrak-api.md), [`07 — Keamanan`](07-keamanan.md)

---

## 1. Tujuan Tahap 10

Tahap 10 mengubah data operasional menjadi bahan pengambilan keputusan pemilik
toko: sembilan laporan bisnis dengan filter tanggal/outlet dan ekspor Excel,
pengaturan toko (termasuk template WhatsApp), manajemen pengguna & outlet,
audit log yang dapat difilter, dan backup manual yang dapat diunduh.

Cakupan yang diselesaikan:

1. **Sembilan laporan bisnis**: pendapatan (harian/bulanan/tahunan), servis,
   merek perangkat, jasa vs sparepart, laba kotor/bersih, pengeluaran,
   produktivitas teknisi, stok, dan nilai inventori.
2. **Filter tanggal + outlet** dengan interpretasi zona waktu outlet dan batas
   rentang 370 hari.
3. **Ekspor xlsx** untuk 8 laporan berbasis rentang (via `format=xlsx`) dengan
   header identitas toko/outlet, rentang, pembuat, dan waktu.
4. **Pengaturan toko**: identitas, tiket/invoice, garansi & struk, provider +
   template WhatsApp — semua melalui `PATCH /settings`.
5. **Manajemen pengguna** (CRUD, reset kata sandi sekali-tampil, penugasan
   outlet) dan **manajemen outlet**.
6. **Audit log viewer** read-only dengan filter modul/aksi/entitas/tanggal dan
   baris rincian before/after.
7. **Backup manual** tenant (dump SQL gzip) dengan metadata, daftar riwayat,
   dan unduh berkas.

---

## 2. File yang Ditambahkan / Diubah

### Backend

```
backend/internal/modules/report/model.go           # DTO 9 laporan + ExportMeta
backend/internal/modules/report/service.go        # Agregasi SQL + resolveScope timezone
backend/internal/modules/report/handler.go        # Handler + filter query + format=xlsx
backend/internal/modules/report/export.go         # Workbook xlsx (excelize)
backend/internal/modules/report/routes.go         # Route /reports/*
backend/internal/modules/settings/model.go        # DTO Settings/WATemplates/Backup
backend/internal/modules/settings/service.go      # Get/Update settings + backup CRUD
backend/internal/modules/settings/dump.go         # TenantDump COPY → gzip (pgx)
backend/internal/modules/settings/handler.go       # Handler settings + backup + unduh
backend/internal/modules/settings/routes.go       # Route /settings*
backend/internal/modules/auditview/service.go     # Query audit_logs + filter + modules
backend/internal/modules/auditview/handler.go      # Handler audit-logs
backend/internal/modules/auditview/routes.go      # Route /audit-logs*
backend/internal/modules/user/{model,dto,repository,service,handler,routes}.go
backend/internal/modules/outlet/{model,service,handler,routes}.go
backend/internal/router/router.go                  # Wiring modul Tahap 10
backend/internal/config/config.go                 # BACKUP_DIR, BACKUP_DSN
backend/internal/database/migrate.go               # ExpectedVersion 18
backend/migrations/000018_backups.{up,down}.sql   # Tabel backups
backend/tests/integration/report_settings_audit_test.go  # Integration test Tahap 10
backend/internal/pkg/version/version.go            # 0.10.0
```

### Frontend

```
frontend/src/lib/api-client.ts                       # apiDownload + saveBlob + DownloadedFile
frontend/src/features/reports/types.ts               # Tipe 9 laporan + filter
frontend/src/features/reports/api.ts                 # Fetcher laporan + exportReportXlsx
frontend/src/features/reports/format.ts              # Format tanggal/durasi laporan
frontend/src/features/reports/components/
  ├── revenue-chart.tsx                              # AreaChart pendapatan (recharts)
  └── report-sections.tsx                            # 9 seksi laporan + loader hook
frontend/src/features/reports/pages/reports-page.tsx # /reports: tab + filter + ekspor
frontend/src/features/settings/types.ts              # Tipe Settings/WATemplates/Backup
frontend/src/features/settings/api.ts                # Client settings + backup
frontend/src/features/settings/format.ts             # formatBytes, label provider WA
frontend/src/features/settings/components/backup-panel.tsx  # Riwayat + buat + unduh
frontend/src/features/settings/pages/settings-page.tsx      # /settings: 5 tab
frontend/src/features/users/types.ts                 # Tipe User/Role
frontend/src/features/users/api.ts                   # Client users/roles/reset/outlets
frontend/src/features/users/components/
  ├── user-form-dialog.tsx                           # Tambah/ubah pengguna
  ├── user-outlets-dialog.tsx                        # PUT /users/:id/outlets
  └── reset-password-dialog.tsx                      # Kata sandi sementara sekali tampil
frontend/src/features/users/pages/user-list-page.tsx # /users: filter + pagination
frontend/src/features/outlets/types.ts               # Tipe Outlet
frontend/src/features/outlets/api.ts                 # Client outlets
frontend/src/features/outlets/components/outlet-form-dialog.tsx  # Tambah/ubah outlet
frontend/src/features/outlets/pages/outlet-list-page.tsx         # /outlets
frontend/src/features/audit/types.ts                 # Tipe AuditLogRow
frontend/src/features/audit/api.ts                   # Client audit-logs
frontend/src/features/audit/pages/audit-log-page.tsx # /audit-logs + baris rincian
frontend/src/app/router.tsx                          # Route Tahap 10
frontend/src/components/layout/nav-items.ts          # Menu Laporan/Pengguna/Outlet/Toko/Audit
frontend/src/components/layout/sidebar.tsx           # Versi 0.10.0
frontend/package.json                                # 0.10.0
```

---

## 3. Perilaku Backend

### 3.1 Sembilan laporan bisnis

Semua endpoint berada di bawah `/api/v1/reports`, hanya untuk pengguna tenant
(super admin mendapat `403`), dengan permission `report.view` — kecuali tiga
laporan keuangan yang membutuhkan `report.profit.view`:

| Endpoint | Isi | Permission |
|---|---|---|
| `GET /reports/revenue` | Pendapatan per periode `daily`/`monthly`/`yearly` + total | `report.view` |
| `GET /reports/services` | Total, selesai, berlangsung, batal, durasi rata-rata, breakdown status, peringkat teknisi | `report.view` |
| `GET /reports/brands` | Distribusi merek perangkat: jumlah servis + pendapatan | `report.view` |
| `GET /reports/service-vs-parts` | Nominal & qty jasa vs sparepart dari invoice terbit | `report.view` |
| `GET /reports/technicians` | Ditugaskan/selesai/aktif, durasi rata-rata, kontribusi pendapatan | `report.view` |
| `GET /reports/stock` | Stok per sparepart, nilai stok, baris stok rendah | `report.view` |
| `GET /reports/profit` | Pendapatan, HPP, laba kotor, pengeluaran, laba bersih, rata-rata invoice | `report.profit.view` |
| `GET /reports/expenses` | Agregasi pengeluaran per kategori | `report.profit.view` |
| `GET /reports/inventory-valuation` | Nilai inventori (cermin laporan stok) | `report.profit.view` |

Perilaku filter yang sama untuk semua laporan:

- `from`/`to` (`YYYY-MM-DD`, inklusif) diinterpretasikan dalam **zona waktu
  outlet**; default `to` = hari ini, `from` = 30 hari sebelumnya.
- Rentang maksimal **370 hari**; `from > to` → `422 VALIDATION_ERROR`.
- `outlet_id` opsional divalidasi terhadap outlet yang dapat diakses aktor;
  tanpa parameter, laporan mengikuti outlet aktif (header `X-Outlet-Id`
  atau outlet pertama aktor).
- `revenue` memvalidasi `period` ∈ `daily|monthly|yearly`; nilai lain →
  `422`.
- Pendapatan hanya menghitung invoice `issued`/`paid` (bukan draft/void);
  `service_amount`/`part_amount` dipisah per tipe item, diskon terpisah.

### 3.2 Ekspor xlsx

- Tambahkan `format=xlsx` pada endpoint laporan berbasis rentang (8 laporan;
  `inventory-valuation` tidak punya ekspor).
- Respons berupa `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
  dengan `Content-Disposition: attachment; filename*=UTF-8''...` dan nama
  berkas `laporan-<jenis>-<from>_<to>.xlsx` (`laporan-stok.xlsx` untuk stok).
- Setiap workbook memuat blok metadata (nama laporan, toko, outlet, rentang,
  dibuat oleh, waktu) lalu sheet data sesuai laporan — hanya kolom yang sudah
  ada di respons JSON; tidak ada kolom sensitif (kredensial, token, data
  pelanggan lintas modul).
- Tanggal dihitung dalam zona waktu outlet yang sama seperti respons JSON,
  sehingga angka xlsx selalu cocok dengan tampilan UI.

### 3.3 Pengaturan toko

- `GET /settings` (permission `settings.view`) mengembalikan seluruh
  `business_settings` level tenant (outlet `IS NULL`): identitas, pajak, zona
  waktu, prefiks/format tiket & invoice, syarat/catatan kaki invoice, garansi
  bawaan, lebar struk, flag teknisi/pelacakan publik, provider WA, template
  WA.
- `PATCH /settings` (permission `settings.manage`) bersifat parsial — hanya
  field yang dikirim yang diubah dalam satu transaksi `FOR UPDATE`.
  Nilai string kosong pada field opsional menghapus kolom (menjadi `NULL`).
- Validasi: `store_name` wajib; `timezone` harus zona waktu valid
  (`Asia/Jakarta` dll.); prefiks tiket/invoice tidak boleh kosong; email
  format valid; `receipt_paper_width` ∈ {58, 80}; `wa_provider` ∈
  {`noop`, `fonnte`, `wablas`}; garansi 0–3650 hari. Invalid → `422` tanpa
  penulisan.
- Setiap perubahan mencatat audit `settings_update` (module `settings`,
  entity `business_settings`) beserta before/after data, dan respons PATCH
  membaca kembali baris **di dalam transaksi yang sama** sehingga nilai yang
  baru ditulis selalu terlihat.
- Perubahan langsung berlaku: nama toko pada invoice baru, prefiks/format
  nomor dokumen, garansi bawaan saat pickup, lebar struk cetak, serta
  provider/template WhatsApp worker.

### 3.4 Audit log viewer

- `GET /audit-logs` (permission `audit.view`, owner saja pada seed RBAC)
  membaca `audit_logs` tenant secara **read-only**, terurut terbaru dulu.
- Filter: `module` (eksak), `action` (ILIKE), `entity_type` (eksak),
  `user_id` (UUID), `from`/`to` (batas hari inklusif), `page`/`per_page`
  (maks 100). Format tanggal salah → `422`.
- Baris berisi `actor_name`, `action`, `module`, `entity_type`, `entity_id`,
  `before_data`, `after_data`, `created_at` — cukup untuk melihat siapa
  mengubah apa dan isinya sebelum/sesudah.
- `GET /audit-logs/modules` mengembalikan daftar modul yang ada di trail
  untuk dropdown filter.
- Super admin tidak dapat membuka audit tenant (403) — modul ini murni
  tenant-scoped.

### 3.5 Manajemen pengguna & outlet

- `GET /users` (paginasi + `q`/`role`/`status`), `POST /users`, `GET/PATCH/
  DELETE /users/:id` (soft delete). Semua dibatasi tenant aktif.
- `POST /users/:id/reset-password` menghasilkan kata sandi sementara yang
  dikembalikan **tepat sekali** dan menandai `must_change_password`; login
  berikutnya memaksa pergantian.
- `PUT /users/:id/outlets` mengganti seluruh penugasan outlet pengguna
  secara atomik; pengguna tanpa outlet tidak dapat mengakses data
  operasional.
- `GET /roles` dan `GET /permissions` (permission `user.view`) memetakan
  peran ke daftar kode permission untuk UI form.
- `GET /outlets`, `POST /outlets`, `PATCH /outlets/:id` mengelola cabang:
  kode (2–20 karakter), nama, alamat, telepon, zona waktu, status aktif.
  Outlet nonaktif tetap terdata namun tidak dipakai untuk transaksi baru.
- Semua operasi manajemen mencatat audit (module `user`/`outlet`).

### 3.6 Backup manual

- Migrasi `000018_backups` membuat tabel `backups` (metadata + status).
- `POST /settings/backups` (permission `settings.manage`) menjalankan dump:
  - Berkas `backup-<tenant_id>-<timestamp>.sql.gz` di folder `BACKUP_DIR`
    (default `backups/`), koneksi khusus melalui `BACKUP_DSN`.
  - Dump memakai `COPY ... TO STDOUT` per tabel (pgx), difilter per tenant:
    21 tabel bereferensi `tenant_id` + 2 tabel tanpa `tenant_id` yang
    memakai predikat join (`notification_reads`, `user_outlets`).
  - Metadata baris: nama berkas, ukuran gzip, jumlah tabel, jumlah baris,
    status `completed`, dibuat oleh. Ukuran pada metadata diambil dari
    berkas final sehingga selalu cocok dengan hasil unduhan.
- `GET /settings/backups` mengembalikan 50 backup terbaru;
  `GET /settings/backups/:id/download` mengalirkan berkas
  `application/gzip`. ID milik tenant lain/tidak ada → `404`.
- Konfigurasi wajib: `BACKUP_DSN` (koneksi database untuk pgx); tanpa itu
  pembuatan backup gagal dengan pesan internal yang jelas.

---

## 4. Frontend

| Route | Halaman | Akses |
|---|---|---|
| `/reports` | 9 tab laporan, filter outlet/tanggal/periode, grafik pendapatan, ekspor xlsx | `report.view` (tab Laba/Pengeluaran/Nilai Inventori hanya `report.profit.view`) |
| `/users` | Daftar pengguna + filter + pagination, dialog tambah/ubah, atur outlet, reset sandi, hapus | `user.view` (aksi `user.manage`) |
| `/outlets` | Daftar outlet, dialog tambah/ubah | `outlet.view` (aksi `outlet.manage`) |
| `/settings` | Tab identitas, tiket & invoice, garansi & struk, WhatsApp, backup | `settings.view` (aksi `settings.manage`) |
| `/audit-logs` | Tabel audit + filter + pagination + baris rincian before/after | `audit.view` |

Perilaku UI:

- **Laporan**: tab tunggal per jenis; filter `from`/`to` default 30 hari
  terakhir; pemilik multi-outlet dapat membandingkan outlet lewat pilihan
  outlet lokal halaman (tanpa mengganti outlet aktif global). Grafik
  pendapatan memakai AreaChart (recharts) dengan tiga garis: total,
  terbayar, belum terbayar. Tombol **Ekspor Excel** mengunduh xlsx sesuai
  tab dan filter aktif; tab Nilai Inventori tidak memiliki tombol ekspor.
- **Pengaturan**: satu form dengan validasi field dari server
  (`ApiError.fieldErrors()`), teks bantuan per field, dan banner
  sukses/gagal. Tab backup terpisah dari form dan hanya tampil untuk
  pemegang `settings.manage`. Pengguna tanpa `settings.manage` melihat
  nilai read-only.
- **Pengguna**: reset kata sandi menampilkan kata sandi sementara sekali
  dengan tombol salin; penugasan outlet memakai checkbox dan langsung
  memperbarui baris. Tombol hapus disembunyikan untuk akun sendiri.
- **Audit**: klik **Rincian** men-expand baris before/after sebagai JSON
  terformat; dropdown modul diambil dari `GET /audit-logs/modules`.
- Unduhan biner (xlsx & gzip) memakai `apiDownload` di `api-client.ts` —
  header auth/outlet dan refresh token yang sama seperti API JSON, lalu
  `saveBlob` menawarkan simpan berkas ke browser.
- Semua halaman memakai komponen bersama (Card, EmptyState, Badge, Alert,
  pagination) dan konvensi styling tahap sebelumnya; navigasi sidebar
  mengaktifkan menu Laporan, Pengguna, Outlet, Toko, dan Audit Log yang
  sebelumnya ditandai Tahap 10/11.

---

## 5. Permission & Keamanan

| Permission | Kemampuan | Owner | Admin | Teknisi | Kasir |
|---|---|---|---|---|---|
| `report.view` | 6 laporan operasional | ✅ | ✅ | ❌ | ✅ |
| `report.profit.view` | Laba, pengeluaran, nilai inventori | ✅ | ❌ | ❌ | ❌ |
| `report.export` | Ekspor laporan (dicadangkan) | ✅ | ✅ | ❌ | ✅ |
| `user.view` / `user.manage` | Lihat / kelola pengguna | ✅ | ✅ | ❌ | ❌ |
| `outlet.view` / `outlet.manage` | Lihat / kelola outlet | ✅ | ✅ | lihat | lihat |
| `settings.view` / `settings.manage` | Lihat / ubah pengaturan + backup | ✅ | ✅ | ❌ | ❌ |
| `audit.view` | Audit log viewer | ✅ | ❌ | ❌ | ❌ |

- Semua endpoint Tahap 10 adalah tenant-scoped; super admin selalu `403`.
  `outlet_id` diperiksa terhadap `OutletIDs` aktor (IDOR lintas outlet
  ditolak `403 OUTLET_FORBIDDEN`).
- Kata sandi sementara hanya dikembalikan sekali melalui HTTPS dan tidak
  pernah ditulis ke audit/log.
- Backup hanya mencakup baris tenant pemanggil; tabel platform (tenants,
  roles, permissions, users) tidak ikut didump — users tenant difilter
  lewat `user_outlets` join. Tenant id diinlinke aman setelah validasi UUID.
- Audit log tidak dapat diubah dari API (read-only), menjaga integritas
  jejak.

---

## 6. Pengujian

Integration test baru di `backend/tests/integration/report_settings_audit_test.go`
(fixture: alur lengkap order → assign teknisi → pakai sparepart → invoice →
pembayaran → expense):

- `TestStage10ReportsAggregatesAndExcelExport` — angka ke-9 laporan dicocokkan
  dengan fixture (pendapatan 400k = jasa 150k + part 250k; HPP 100k; laba
  bersih 250k setelah expense 50k; stok 4 unit; kontribusi teknisi), xlsx
  revenue & stok ter-download sebagai container zip valid (`PK`), period
  invalid dan rentang terbalik ditolak.
- `TestStage10ReportPermissionGates` — admin tetap dapat laporan operasional
  namun `403` untuk laba/nilai inventori/audit; super admin `403` untuk semua
  laporan tenant.
- `TestStage10SettingsAuditAndManualBackup` — GET/PATCH settings, validasi
  (timezone invalid, prefiks kosong), nama toko baru muncul pada invoice
  berikutnya, audit `settings_update` dapat difilter per modul/entitas,
  daftar modul, backup dibuat (≥ 20 tabel, baris > 0), daftar backup, unduh
  gzip dengan ukuran yang cocok metadata, dan ID tidak dikenal → `404`.
- `TestMigrationUpDownUp` (schema test) — migrasi 18 naik/turun/naik bersih.

Hasil:

```bash
cd backend
go build ./...
go vet ./...
go test ./... -count=1
# PASS — seluruh paket, termasuk integration suite

cd frontend
npm run typecheck
npm run build
# PASS — hanya peringatan ukuran chunk > 500 kB (sama seperti Tahap 9)
```

`docs/openapi.yaml` diperbarui menjadi versi **0.10.0** dengan endpoint
`/reports/*`, `/settings*`, `/audit-logs*`, `/users*`, `/roles`, `/permissions`,
dan `/outlets*`.

---

## 7. Batasan & Langkah Berikutnya

Batasan yang disengaja pada tahap ini:

- Ekspor hanya **xlsx**; PDF laporan belum tersedia (kontrak API menyebut
  keduanya; PDF menyusul saat kebutuhan nyata muncul).
- Laporan `damage-types` pada kontrak API **belum diimplementasikan** karena
  skema belum punya taksonomi jenis kerusakan; laporan `brands` menjadi
  proxy distribusi perangkat. Akan ditambahkan bersama model data kerusakan.
- `inventory-valuation` masih cermin laporan stok (JSON saja, tanpa xlsx)
  dan belum membedakan metode valuasi (FIFO/rata-rata bergerak).
- Backup **restore** belum tersedia dari UI — restore di staging dijadwalkan
  pada Tahap 11 (uji pemulihan).
- `logo_path` hanya dibaca; upload logo belum ada (menunggu storage file).
- Retensi/purge audit log belum dapat dikonfigurasi; tabel tumbuh tanpa
  batas waktu.
- Laporan dieksekusi langsung dari SQL agregat tanpa tabel summary/queue
  export; cocok untuk skala MVP, perlu optimasi bila volume transaksi besar.
- `report.export` belum dipisahkan dari `report.view` (endpoint xlsx cukup
  `report.view`/`report.profit.view`).

Tahap berikutnya adalah **Tahap 11 — Testing & Security Review**: integration
test alur kritis lintas modul, E2E Playwright, uji konkurensi & beban, RLS
PostgreSQL, audit keamanan (`gosec`, `npm audit`, OWASP ZAP), dan uji restore
backup di staging.
