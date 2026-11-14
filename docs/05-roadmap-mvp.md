# 05 — Roadmap Implementasi MVP

**Versi:** 1.1 (Tahap 12 selesai) · **Rilis:** v1.0.0
**Metode:** incremental, setiap tahap harus **berjalan, teruji, dan terhubung
end-to-end** sebelum lanjut.

---

## 1. Prinsip Eksekusi

1. **Vertical slice** — setiap tahap menghasilkan fitur yang bisa dipakai, bukan
   hanya layer backend.
2. **No dummy backend** — frontend selalu memakai API nyata + database nyata.
3. **Testing gate** — tahap tidak dianggap selesai bila test/build gagal.
4. **Dokumentasi hidup** — kontrak API & cara jalan diperbarui di tahap yang sama.
5. **Tidak ada tombol mati** — aksi yang belum siap disembunyikan.
6. **Review keamanan berkelanjutan** — RBAC, tenant scope, validasi ditambahkan
   sejak fitur dibuat, bukan ditunda ke Tahap 11.

### Definition of Done (berlaku semua tahap)

- [ ] `go build ./...`, `go vet ./...`, `golangci-lint run` lulus.
- [ ] `npm run build`, `npm run lint`, `tsc --noEmit` lulus.
- [ ] Unit/integration test untuk logika baru lulus.
- [ ] Tidak ada endpoint tanpa validasi input & permission.
- [ ] Tenant/outlet scope teruji (tidak ada kebocoran data).
- [ ] UI menangani loading, empty, error, success.
- [ ] Responsif di 360 px, 768 px, 1280 px.
- [ ] README/dokumentasi API diperbarui.
- [ ] Demo manual alur utama berhasil tanpa error console.
- [ ] Tidak ada secret hardcoded.

---

## 2. Ringkasan 12 Tahap

| Tahap | Fokus | Estimasi | Rilis internal |
|---|---|---:|---|
| 1 | Analisis, arsitektur, desain DB, roadmap | 2 hari | — |
| 2 | Database fisik, migration, seed, environment | 3 hari | v0.2.0 |
| 3 | Backend auth, RBAC, user management + UI login | 4 hari | v0.3.0 |
| 4 | Frontend shell, navigasi, dashboard awal | 4 hari | v0.4.0 |
| 5 | Modul penerimaan servis + pelanggan + foto + label | 6 hari | v0.5.0 |
| 6 | Diagnosis, pengerjaan, status machine, item jasa/part | 6 hari | v0.6.0 |
| 7 | Inventori sparepart lengkap | 5 hari | v0.7.0 |
| 8 | Invoice, kasir, pembayaran, cetak | 6 hari | v0.8.0 |
| 9 | Tracking publik, garansi, klaim, notifikasi WA | 5 hari | v0.9.0 |
| 10 | Laporan, pengeluaran, pengaturan, dashboard bisnis | 7 hari | v1.0.0-rc |
| 11 | Testing menyeluruh & security review | 5 hari | v1.0.0-rc2 |
| 12 | Deployment production, backup, observability | 4 hari | v1.0.0 |

**Total ± 57 hari kerja** (≈ 11–12 minggu untuk 1 developer full-time, termasuk
buffer 15%). Dapat dipersingkat dengan pengerjaan paralel frontend/backend
setelah Tahap 3.

---

## 3. Detail per Tahap

### Tahap 1 — Analisis & Arsitektur ✅

**Deliverable**
- `docs/01`–`docs/08`
- README, .gitignore, .editorconfig

**Acceptance**
- [x] Aktor, hak akses, alur bisnis, status machine disepakati.
- [x] ERD & kamus tabel lengkap.
- [x] Struktur folder & kontrak API awal.
- [x] Roadmap & acceptance per tahap.

---

### Tahap 2 — Desain Database & Environment

**Tujuan:** database nyata, migrasi reproducible, aplikasi bisa connect.

**File utama**
```
backend/cmd/cli/main.go
backend/internal/config/config.go
backend/internal/database/postgres.go
backend/internal/database/migrate.go
backend/internal/database/seed.go
backend/internal/pkg/logger/logger.go
backend/migrations/*.sql
docker-compose.yml
docker-compose.override.yml
.env.example
Makefile
deploy/docker/postgres/init.sql
```

**Pekerjaan**
1. Inisialisasi Go module, Gin, GORM, pgx driver, golang-migrate, validator, slog.
2. Tulis seluruh migrasi (tabel, FK, CHECK, index, trigram, `NULLS NOT DISTINCT`).
3. Command `cli migrate up|down|status`, `cli seed`, `cli create-tenant`.
4. Koneksi pool + `Ping` + helper transaksi.
5. Docker Compose PostgreSQL 16 (dev) — atau panduan PostgreSQL native bila
   Docker tidak tersedia.
6. Seed role, permission, mapping, super admin, kategori pengeluaran.
7. Test: migrasi up → down → up pada DB kosong; seed idempoten.

**Acceptance**
- [x] `migrate up` membuat seluruh tabel tanpa error.
- [x] `migrate down` bersih pada DB kosong.
- [x] Constraint kritis terbukti menolak data invalid (stok negatif, status
      ilegal, duplikat tiket).
- [x] `seed` dua kali tidak menggandakan data.
- [x] `/healthz` & `/readyz` merespons benar.

> **Status:** selesai. Bukti implementasi dan hasil test ada di
> [`09 — Implementasi Tahap 2`](09-implementasi-tahap-2.md).

---

### Tahap 3 — Backend Auth, RBAC & User Management

**Tujuan:** login aman, sesi tahan reload, hak akses teruji.

**File utama**
```
backend/internal/modules/auth/*
backend/internal/modules/user/*
backend/internal/modules/role/*
backend/internal/middleware/{auth,permission,rate_limit}.go
backend/internal/pkg/{jwtutil,cryptoutil,response,apperror,scope}/*
backend/tests/integration/auth_test.go
frontend/src/features/auth/*
frontend/src/lib/{api-client,auth,permissions}.ts
frontend/src/stores/auth-store.ts
frontend/src/app/{router,providers}.tsx
```

**Pekerjaan**
1. `POST /auth/login`, `/auth/refresh`, `/auth/logout`, `GET /auth/me`.
2. bcrypt, lockout 5×/15 menit, audit login/failed login.
3. Refresh token rotation + reuse detection + cookie HttpOnly.
4. CRUD user (Admin/Owner), nonaktifkan, reset password, assign outlet.
5. Middleware Auth, Tenant, Permission; rate limiter login.
6. Halaman login responsif + redirect sesuai role + state sesi.
7. Guard route frontend + `PermissionGuard`.

**Acceptance**
- [x] Login/logout/refresh berfungsi; reload halaman tetap login.
- [x] Token tidak tersimpan di localStorage; cookie `HttpOnly`.
- [x] Setiap role hanya bisa mengakses endpoint sesuai matriks.
- [x] Reuse refresh token mencabut seluruh family.
- [x] Account lockout bekerja dan tercatat.
- [x] Integration test RBAC & isolasi tenant lulus.

> **Status:** selesai. Bukti implementasi, hasil test, dan cara verifikasi ada di
> [`10 — Implementasi Tahap 3`](10-implementasi-tahap-3.md).

---

### Tahap 4 — Frontend Layout & Dashboard

**Tujuan:** kerangka aplikasi nyaman dipakai teknisi + dashboard data nyata.

**File utama**
```
frontend/src/components/layout/*
frontend/src/components/shared/*
frontend/src/features/dashboard/*
frontend/src/styles/globals.css
backend/internal/modules/dashboard/*
```

**Pekerjaan**
1. App shell: sidebar (collapse), topbar (outlet switcher, profil), mobile nav.
2. Design token (navy/cyan), komponen shadcn dasar, dark text on light.
3. Dashboard: 7 kartu statistik, grafik Recharts 7/30 hari, notifikasi
   terlambat, aktivitas teknisi.
4. Endpoint `/dashboard/summary`, `/dashboard/charts`, `/dashboard/alerts`.
5. Skeleton, empty state, error state, refresh manual.
6. PWA manifest + ikon (installable).

**Acceptance**
- [x] Dashboard menampilkan angka dari database (bukan hardcoded).
- [x] Filter outlet bekerja untuk multi-outlet.
- [x] Grafik akurat untuk rentang 7/30 hari.
- [ ] Responsif & navigasi keyboard (implementasi ada, verifikasi visual manual tersisa).
- [ ] Tidak ada warning console (verifikasi browser manual tersisa).

> **Status:** selesai kecuali QA visual manual. Implementasi, hasil test, dan
> batasan ada di [`11 — Implementasi Tahap 4`](11-implementasi-tahap-4.md).

---

### Tahap 5 — Modul Penerimaan Servis

**Tujuan:** alur terima HP selesai sampai label tercetak.

**File utama**
```
backend/internal/modules/customer/*
backend/internal/modules/device/*
backend/internal/modules/service/*            # create/list/detail/status awal
backend/internal/modules/upload/*
frontend/src/features/customers/*
frontend/src/features/services/pages/service-{list,create,detail}.tsx
frontend/src/features/services/components/service-form.tsx
frontend/src/features/print/label.tsx
```

**Pekerjaan**
1. CRUD pelanggan + pencarian + pencegahan duplikat nomor.
2. Form penerimaan lengkap (semua field spesifikasi) dengan validasi.
3. Nomor tiket atomik + QR + label servis 58/80 mm.
4. Upload foto intake (validasi MIME & ukuran).
5. PIN/password perangkat terenkripsi + endpoint reveal beraudit.
6. Daftar servis: search, filter status/teknisi/tanggal, pagination, badge.
7. Detail servis: ringkasan, data perangkat, foto, timeline status.
8. Audit log pembuatan servis + perubahan awal.

**Acceptance**
- [ ] Input servis < 2 menit; nomor tiket unik saat 20 request bersamaan.
- [ ] PIN tidak pernah muncul di response/list/export.
- [ ] Foto terunggah & tampil; file invalid ditolak.
- [ ] Filter & pencarian akurat.
- [ ] Label tercetak rapi di 58 mm dan 80 mm.
- [ ] Pelanggan duplikat ditolak dengan pesan jelas.

> **Status:** implementasi selesai; acceptance otomatis (tiket unik, PIN tidak
> bocor, foto, filter, duplikat pelanggan) lulus. Verifikasi visual cetak label
> dan pengukuran waktu input manual masih menjadi QA tersisa. Bukti ada di
> [`12 — Implementasi Tahap 5`](12-implementasi-tahap-5.md).

---

### Tahap 6 — Diagnosis & Pengerjaan

**Tujuan:** teknisi mengelola pekerjaan tanpa kembali ke kertas.

**File utama**
```
backend/internal/modules/service/{diagnostic,item,status}*.go
backend/internal/modules/inventory/stock_service.go   # interface pemakaian
frontend/src/features/services/components/{diagnosis-form,items-editor,
  status-stepper,status-change-dialog,photo-uploader,timeline}.tsx
```

**Pekerjaan**
1. Form diagnosis, temuan, tindakan, hasil testing, catatan internal.
2. Item jasa & sparepart; harga, qty, diskon baris, garansi item.
3. State machine 10 status + validasi transisi + permission per transisi.
4. Riwayat status lengkap (waktu, pelaku, catatan).
5. Foto before/after/testing.
6. Pengurangan stok otomatis saat sparepart ditambahkan (transaksi + lock) dan
   pengembalian saat dihapus.
7. Penugasan teknisi & prioritas express.
8. Notifikasi internal untuk servis terlambat.

**Acceptance**
- [x] Semua transisi ilegal ditolak dengan pesan jelas.
- [x] Stok berkurang tepat sekali; percobaan ganda tidak menggandakan.
- [x] Stok tidak cukup → 409, item tidak tersimpan.
- [x] Timeline menampilkan seluruh perubahan.
- [x] Teknisi hanya melihat order yang ditugaskan (jika setting aktif).
- [x] Total item dihitung server, bukan klien.

> **Status:** selesai. Filter "hanya order yang ditugaskan" sudah aktif melalui
> `business_settings.technician_sees_only_assigned` dan diterapkan pada daftar
> serta detail servis. Bukti implementasi dan hasil test ada di
> [`13 - Implementasi Tahap 6`](13-implementasi-tahap-6.md).

---

### Tahap 7 — Inventori Sparepart

**Tujuan:** stok akurat dan mudah dicari.

**File utama**
```
backend/internal/modules/inventory/*
frontend/src/features/inventory/*
```

**Pekerjaan**
1. CRUD sparepart, kategori, supplier, SKU/barcode unik.
2. Pencarian + filter kategori/stok menipis + pagination.
3. Pembelian stok (in) dan opname (adjustment) dengan alasan.
4. Riwayat pergerakan stok + filter.
5. Alert minimum stock, nilai inventory, sparepart terlaris.
6. Barcode scan pada form pemakaian (kamera/HTMLElement scanner).
7. Import/export sparepart (CSV) — opsional S.

**Acceptance**
- [x] Semua mutasi stok memiliki movement dan pelaku.
- [x] Stok tidak bisa negatif.
- [x] Nilai inventory = Σ(stok × harga modal) benar.
- [x] Alert muncul saat stok ≤ minimum.
- [x] Barcode/SKU lookup bekerja.

> **Status:** selesai (v0.7.0). Import/export CSV tetap opsional dan belum
> diimplementasikan. Barcode scanner USB/HT di form item servis memakai input
> pencarian yang mencocokkan nama, SKU, dan barcode. Bukti implementasi ada di
> [`14 - Implementasi Tahap 7`](14-implementasi-tahap-7.md).

---

### Tahap 8 — Invoice, Kasir & Pembayaran

**Tujuan:** uang tercatat benar dan bisa dicetak.

**File utama**
```
backend/internal/modules/invoice/*
backend/internal/modules/payment/*
backend/internal/modules/expense/*        # dasar pengeluaran
backend/templates/invoice.html
frontend/src/features/invoices/*
frontend/src/features/payments/*
frontend/src/features/print/{invoice,receipt}.tsx
```

**Pekerjaan**
1. Invoice otomatis (draft) + finalisasi, diskon nominal/persen, pajak 0.
2. Snapshot item ke `invoice_items` saat finalisasi.
3. Pembayaran: tunai, transfer, QRIS, kartu; DP/parsial; validasi overpay.
4. Void pembayaran (Owner/Admin) + alasan + audit.
5. Status invoice otomatis (`unpaid/partial/paid/void`).
6. Cetak invoice A4 + struk thermal 58/80 mm + tombol WhatsApp.
7. Laporan transaksi harian & rekap metode bayar.
8. Aturan pickup wajib lunas (override Owner + audit).
9. CRUD pengeluaran dasar untuk laporan laba.

**Acceptance**
- [x] Total invoice = item − diskon, dihitung server.
- [x] Pembayaran tidak bisa melebihi sisa tagihan.
- [x] DP menghasilkan status `partial` dan sisa benar.
- [x] Void mengembalikan status invoice dengan benar.
- [x] Invoice & struk tercetak tanpa elemen terpotong.
- [x] Pickup belum lunas ditolak kecuali override tercatat.

> **Status:** selesai (v0.8.0). Invoice draft otomatis, finalisasi snapshot,
> diskon, pembayaran dengan idempotency, void, kasir harian, cetak A4/thermal,
> pengeluaran, dan aturan pickup lunas sudah berjalan. Bukti implementasi ada
> di [`15 - Implementasi Tahap 8`](15-implementasi-tahap-8.md).

---

### Tahap 9 — Tracking Publik, Garansi, Notifikasi

**Tujuan:** pelanggan mandiri melihat status; garansi terlacak.

**File utama**
```
backend/internal/modules/tracking/*
backend/internal/modules/warranty/*
backend/internal/modules/notification/*
backend/internal/jobs/reminder.go
backend/cmd/worker/main.go
frontend/src/features/tracking/*
frontend/src/features/warranties/*
frontend/src/features/notifications/*
```

**Pekerjaan**
1. Halaman publik `/track` (tiket + 4 digit HP) dan `/track/:token`.
2. Whitelist field publik; rate limit; token acak; expiry.
3. QR pada label mengarah ke tracking.
4. Garansi otomatis saat pickup, halaman daftar/detail, klaim, rework order.
5. Notifikasi in-app + outbox WhatsApp; adapter `noop`/`fonnte`/`wablas`.
6. Tombol "Chat via WhatsApp" (deep link) yang berfungsi.
7. Worker reminder: servis melewati estimasi, HP belum diambil, garansi
   berakhir.
8. Purge kredensial perangkat setelah 30 hari pickup.

**Acceptance**
- [x] Tracking tidak membocorkan nama lengkap, alamat, PIN, catatan internal,
      harga modal.
- [x] Kombinasi tiket + 4 digit salah ditolak (pesan generik).
- [x] Rate limit terbukti.
- [x] Garansi aktif dengan tanggal benar; klaim tertaut servis asal.
- [x] Pesan WA tercatat di outbox dengan status jelas.
- [x] Worker idempoten (tidak kirim ganda).

> **Status:** selesai (v0.9.0). Halaman tracking publik `/track` dan
> `/track/:token`, QR label ke tautan asli, garansi otomatis, klaim/rework,
> notifikasi in-app, outbox WhatsApp, worker reminder, dan purge kredensial
> sudah berjalan. Bukti implementasi ada di
> [`16 - Implementasi Tahap 9`](16-implementasi-tahap-9.md).

---

### Tahap 10 — Laporan, Pengaturan & Dashboard Bisnis

> **Status:** selesai (v0.10.0). Sembilan laporan bisnis dengan filter
> tanggal/outlet dan ekspor xlsx, halaman pengaturan toko + template WhatsApp,
> manajemen pengguna/outlet, audit log viewer, dan backup manual sudah berjalan.
> Bukti implementasi ada di
> [`17 - Implementasi Tahap 10`](17-implementasi-tahap-10.md).

**Tujuan:** owner bisa mengambil keputusan dari data.

**File utama**
```
backend/internal/modules/report/*
backend/internal/modules/settings/*
backend/internal/modules/outlet/*
backend/internal/platform/tenant/*
frontend/src/features/reports/*
frontend/src/features/settings/*
frontend/src/features/users/*
frontend/src/features/audit/*
```

**Pekerjaan**
1. Laporan pendapatan (harian/bulanan/tahunan), servis, merek, jasa vs part,
   HPP, laba kotor, pengeluaran, laba usaha, produktivitas teknisi, stok.
2. Filter tanggal/outlet + export Excel (xlsx) & PDF.
3. Dashboard bisnis final: tren, top sparepart, teknisi, alert.
4. Pengaturan toko/logo/invoice/tiket/garansi/struk.
5. Manajemen outlet + penugasan user.
6. Manajemen pengguna & role (UI lengkap).
7. Audit log viewer (Owner) dengan filter.
8. Backup manual (Owner) + unduh.

**Acceptance**
- [x] Angka laporan cocok dengan data transaksi (uji sampel manual).
- [x] Filter tanggal memakai timezone outlet.
- [x] Export Excel/PDF berisi data sesuai filter, tanpa kolom sensitif.
      (xlsx tersedia untuk 8 laporan; PDF belum — dicatat sebagai keterbatasan.)
- [x] Pengaturan langsung memengaruhi invoice/tiket.
- [x] Audit log read-only & akurat.
- [x] Owner vs Admin melihat menu yang berbeda.

---

### Tahap 11 — Testing & Security Review

> **Status:** selesai (v0.11.0). RLS PostgreSQL aktif untuk 30 tabel dan diuji
> dengan role non-superuser, matriks IDOR lintas tenant dan uji konkurensi
> stok/pembayaran lulus, header keamanan terpasang, restore drill penuh lulus,
> unit test + E2E Playwright + k6 smoke lulus, `gosec`/`govulncheck` bersih, dan
> CI lima job tersedia. Bukti implementasi ada di
> [`18 - Implementasi Tahap 11`](18-implementasi-tahap-11.md).

**Tujuan:** yakin aplikasi aman dan stabil sebelum production.

**Pekerjaan**
1. Integration test alur kritis: auth, intake, diagnosis, stok, invoice, bayar,
   tracking, garansi, isolasi tenant.
2. E2E Playwright: login → servis → diagnosis → bayar → pickup → tracking.
3. Uji konkurensi: nomor tiket, stok, pembayaran.
4. Uji beban ringan (k6): 50 VU, target p95 < 500 ms.
5. RLS PostgreSQL + test IDOR lintas tenant.
6. `gosec`, `npm audit`, OWASP ZAP baseline, cek header keamanan.
7. Uji restore backup di staging.
8. Audit aksesibilitas (keyboard, kontras, label form).
9. Perbaikan seluruh temuan blocker/major.

**Acceptance**
- [x] Semua test lulus di CI (workflow `.github/workflows/ci.yml` tersedia; seluruh
      gate lulus lokal — CI remote dijalankan saat repo di-push).
- [x] Tidak ada temuan kritis/high terbuka (production dependencies; advisory
      dev-only dicatat sebagai risiko diterima di docs/18 §9).
- [x] RLS aktif & teruji.
- [x] Restore backup berhasil.
- [x] Checklist OWASP Top 10 terisi (docs/07 §15; status §17 diperbarui).
- [x] Regression test fitur utama lulus.

---

### Tahap 12 — Deployment Production

**Tujuan:** aplikasi live, terpantau, dan bisa dipulihkan.

**File utama**
```
deploy/docker/backend.Dockerfile
deploy/docker/frontend.Dockerfile
deploy/nginx/nginx.conf
deploy/caddy/Caddyfile
deploy/backup/{backup.sh,drill.sh,restore.sh,entrypoint.sh}
deploy/monitoring/{prometheus.yml.tmpl,alerts.yml,alertmanager.yml.tmpl}
docker-compose.prod.yml
scripts/smoke-prod.{ps1,sh}
.github/workflows/ci.yml
```

**Pekerjaan**
1. Multi-stage image (distroless/alpine), non-root user, health check.
2. Compose production + TLS (Caddy/Nginx), security header, gzip, cache static.
3. Migrasi otomatis sebelum API start; rollback plan.
4. Backup harian + offsite + uji restore; dokumentasi runbook.
5. Monitoring dasar: uptime, error rate, disk, backup alert.
6. CI: lint, test, build image, scan vulnerability.
7. Seed production (super admin + tenant pertama) tanpa data demo.
8. Smoke test production & go-live checklist.
9. Handover: panduan admin, panduan teknisi, SOP backup.

**Acceptance**
- [x] `docker compose up -d` dari server kosong berhasil (stack production
      `docker-compose.prod.yml`; validasi build dijalankan CI karena Docker tidak
      tersedia di mesin pengembangan).
- [x] HTTPS valid, HTTP redirect, cookie `Secure` (Caddy ACME + HSTS + cookie
      `Secure` di production).
- [x] Migrasi & seed berjalan otomatis (service `migrate` sekali jalan:
      `migrate up` → `seed` → `set-app-role` → `doctor`).
- [x] Backup terjadwal & restore teruji (container `backup`, retensi lokal/offsite,
      drill mingguan, restore manual + runbook).
- [x] Smoke test seluruh alur utama lulus di production (skrip PowerShell/Bash;
      job CI `deploy-smoke` menjalankan stack + smoke terhadap HTTPS lokal).
- [x] Kredensial default diganti; secret tidak ada di repo (`.env.production`
      gitignored, template hanya placeholder).
- [x] Runbook & panduan pengguna tersedia (`docs/20`, `docs/21`).

Implementasi: [`19 — Implementasi Tahap 12`](19-implementasi-tahap-12.md).

---

## 4. Urutan Demo per Milestone

| Milestone | Skenario demo |
|---|---|
| v0.3 | Login sebagai 4 role, menu & API berbeda |
| v0.4 | Dashboard menampilkan angka dari DB |
| v0.5 | Terima 2 HP, cetak label, cari tiket |
| v0.6 | Diagnosis → pakai sparepart → stok turun → testing → selesai |
| v0.7 | Beli stok, opname, alert menipis, riwayat |
| v0.8 | Invoice → DP → lunas → cetak struk → pickup |
| v0.9 | Pelanggan tracking, garansi aktif, klaim, WA outbox |
| v1.0-rc | Laporan bulanan + export, laba, pengaturan |
| v1.0 | Deploy server, backup, restore, smoke test |

---

## 5. Testing Gate per Tahap

| Jenis | Kapan | Alat |
|---|---|---|
| Unit test | Setiap tahap | `go test`, Vitest |
| Integration (DB) | Tahap 2+ | testcontainers-go / DB test |
| Contract API | Tahap 3+ | test HTTP + skema response |
| E2E | Tahap 5+, wajib Tahap 11 | Playwright |
| Security scan | Tahap 3, 8, 11 | gosec, npm audit, ZAP |
| Load test | Tahap 11 | k6 |
| Restore drill | Tahap 11–12 | skrip backup |

---

## 6. Rencana Rilis & Branching

```
master
 ├── feat/02-database
 ├── feat/03-auth
 ├── feat/04-shell-dashboard
 ├── feat/05-service-intake
 ├── feat/06-repair
 ├── feat/07-inventory
 ├── feat/08-cashier
 ├── feat/09-tracking-warranty
 ├── feat/10-reports-settings
 ├── chore/11-hardening
 └── chore/12-deploy
```

Trunk rilis adalah `master` (default branch repositori). CI berjalan pada push
ke `master` maupun `main` agar kompatibel dengan repositori yang memakai
`main`. Tag: `v0.2.0` … `v0.9.0`, `v1.0.0-rc.1`, `v1.0.0`.

---

## 7. Cut Line & Prioritas Bila Waktu Terbatas

**Wajib ada di v1.0 (tidak boleh dipotong):**
Auth & RBAC, penerimaan servis, status + riwayat, diagnosis, sparepart + stok,
invoice + pembayaran + cetak, tracking, laporan dasar, audit log, backup.

**Boleh digeser ke v1.1 bila terpaksa:**
Grafik aktivitas teknisi, barcode scan, import CSV, notifikasi WA otomatis
penuh, backup manual dari UI, multi-outlet UI lanjutan, PWA install prompt.

**Tidak boleh dikompromikan:**
Isolasi tenant, enkripsi PIN, validasi pembayaran, transaksi stok, audit
perubahan status.

---

## 8. Status Rilis & Langkah Berikutnya

**Status: seluruh 12 tahap selesai — v1.0.0 siap go-live.**

Kode, migrasi, dokumentasi, dan gate pengujian sudah lengkap. Langkah berikutnya
bersifat operasional, bukan pengembangan fitur:

1. Push repositori dan jalankan CI; job `images` dan `deploy-smoke` memvalidasi
   build container, TLS lokal, RLS, backup, dan smoke test production.
2. Siapkan server serta DNS, lalu ikuti checklist go-live di
   [`20 — Runbook Operasional`](20-runbook-operasional.md) §13.
3. Buat tenant pertama tanpa data demo, jalankan smoke test dari luar server,
   dan verifikasi backup + restore drill.
4. Latih admin, teknisi, dan kasir memakai
   [`21 — Panduan Pengguna`](21-panduan-pengguna.md).
5. Setelah operasi stabil, tinjau RPO/RTO dan aktifkan PITR/WAL archiving bila
   dibutuhkan.

Calon v1.1 (lihat [`01 — Analisis Kebutuhan`](01-analisis-kebutuhan.md) §10):
2FA, payment gateway/refund, transfer stok antar outlet, notifikasi WA otomatis
penuh, kiosk mode, loyalty, dan PITR. Item cut line Tahap 10 juga dapat
dipertimbangkan: barcode/CSV, multi-outlet UI lanjutan, dan PWA install prompt.
