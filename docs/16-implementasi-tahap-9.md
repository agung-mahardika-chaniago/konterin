# 16 — Implementasi Tahap 9: Tracking Publik, Garansi & Notifikasi

**Status:** selesai ✅ **Versi aplikasi:** 0.9.0 ✅ **Tanggal:** 9 November 2026
**Acuan:** [`05 — Roadmap MVP`](05-roadmap-mvp.md) Tahap 9, [`06 — Kontrak API`](06-kontrak-api.md), [`07 — Keamanan`](07-keamanan.md)

---

## 1. Tujuan Tahap 9

Tahap 9 membuat pelanggan dapat memantau progres servis tanpa menghubungi
toko, memastikan garansi dan klaim tercatat setelah perangkat diambil, serta
menyediakan notifikasi staf dan outbox WhatsApp yang dapat diaudit.

Cakupan yang diselesaikan:

1. **Halaman publik `/track`** dengan kombinasi nomor tiket + 4 digit terakhir
   nomor HP, dan **`/track/:token`** untuk tautan/QR langsung.
2. **Whitelist field publik**, pesan error generik, rate limit, token acak
   32 karakter, dan masa berlaku token.
3. **QR label servis** mengarah ke tautan tracking asli dari API.
4. **Garansi otomatis saat pickup**, daftar/detail garansi, klaim, dan order
   rework yang tertaut.
5. **Notifikasi in-app** dengan status baca per pengguna dan broadcast tenant.
6. **Outbox WhatsApp** dengan adapter `noop`/`fonnte`/`wablas`, status jelas,
   retry aman, dan deep link `wa.me`.
7. **Worker** untuk outbox, pengingat servis/pickup/garansi, dan purge
   kredensial 30 hari setelah pickup.
8. **Pengujian integrasi** untuk privasi tracking, rate limit, garansi/klaim,
   rework, notifikasi, outbox, dan worker idempoten.

---

## 2. File yang Ditambahkan / Diubah

### Backend

```
backend/cmd/worker/main.go                              # Entry point worker + interval env
backend/internal/jobs/reminder.go                       # Reminder, outbox drain, purge kredensial
backend/internal/modules/tracking/model.go              # DTO PublicTracking & StoreInfo
backend/internal/modules/tracking/service.go            # Lookup tiket/token, whitelist, expiry
backend/internal/modules/tracking/handler.go            # Handler publik
backend/internal/modules/tracking/routes.go             # Route /public/*
backend/internal/modules/warranty/model.go              # DTO warranty, claim, rework
backend/internal/modules/warranty/service.go            # Garansi pickup, klaim, transisi, rework
backend/internal/modules/warranty/handler.go            # Handler warranty/claim
backend/internal/modules/warranty/routes.go             # Route warranty/claim
backend/internal/modules/notification/model.go          # DTO notifikasi, outbox, template, status
backend/internal/modules/notification/service.go        # Broadcast, read receipts, enqueue, outbox
backend/internal/modules/notification/providers.go      # Adapter noop/fonnte/wablas
backend/internal/modules/notification/handler.go        # Handler notifikasi & outbox
backend/internal/modules/notification/routes.go         # Route notifikasi & WhatsApp
backend/internal/modules/service/orders.go              # GET /service-orders/:id/public-link
backend/internal/modules/service/status.go              # Hook pickup + purge_at
backend/internal/modules/service/model.go               # PublicLink, public token DTO
backend/internal/modules/service/handler.go             # Handler public-link
backend/internal/modules/service/routes.go              # Route public-link
backend/internal/router/router.go                       # Wiring modul, hook, rate limiter publik
backend/internal/config/config.go                       # Token WA & base URL
backend/internal/database/migrate.go                    # ExpectedVersion 17
backend/migrations/000017_tracking_warranty_notifications.*.sql
backend/tests/integration/tracking_warranty_test.go     # Integration test Tahap 9
backend/tests/integration/schema_test.go                # Migrasi up/down/up 17
backend/internal/pkg/version/version.go                 # 0.9.0
```

### Frontend

```
frontend/src/features/tracking/types.ts                       # Tipe payload tracking publik
frontend/src/features/tracking/api.ts                         # trackByTicket, trackByToken
frontend/src/features/tracking/components/
  ├── public-page-shell.tsx                                   # Shell publik tanpa auth
  └── tracking-result.tsx                                     # Progres, perangkat, garansi
frontend/src/features/tracking/pages/
  ├── tracking-lookup-page.tsx                                # /track + prefill ?ticket=
  └── tracking-detail-page.tsx                                # /track/:token
frontend/src/features/warranties/types.ts                     # Tipe warranty, claim, rework
frontend/src/features/warranties/api.ts                       # Client API + deep link WhatsApp
frontend/src/features/warranties/format.ts                    # Label status, tanggal, sisa hari
frontend/src/features/warranties/components/
  ├── status-badges.tsx                                       # Badge garansi/tipe/klaim
  ├── claim-create-dialog.tsx                                 # Form klaim baru
  └── claim-resolve-dialog.tsx                                # Selesai/tolak + resolusi
frontend/src/features/warranties/pages/
  ├── warranty-list-page.tsx                                  # Tab garansi & klaim, filter
  ├── warranty-detail-page.tsx                                # Detail + riwayat klaim
  └── claim-detail-page.tsx                                   # Transisi klaim + rework
frontend/src/features/notifications/types.ts                  # Tipe notifikasi & outbox
frontend/src/features/notifications/api.ts                    # Client API inbox/outbox
frontend/src/features/notifications/format.ts                 # Status WA, template, waktu relatif
frontend/src/features/notifications/components/
  ├── notification-icon.tsx                                   # Ikon per tipe notifikasi
  └── notification-menu.tsx                                   # Bell dropdown + unread count
frontend/src/features/notifications/pages/
  ├── notification-list-page.tsx                              # Inbox semua/belum dibaca
  └── whatsapp-outbox-page.tsx                                # Riwayat & retry outbox
frontend/src/features/print/print-label-modal.tsx             # Ambil public-link asli
frontend/src/features/print/thermal-label.tsx                 # QR tracking nyata + fallback tiket
frontend/src/features/services/api.ts                         # getServiceOrderPublicLink
frontend/src/features/services/types.ts                       # PublicLink
frontend/src/app/router.tsx                                   # Route publik & Tahap 9
frontend/src/components/layout/topbar.tsx                     # NotificationMenu
frontend/src/components/layout/nav-items.ts                   # Menu Garansi/Notifikasi/Outbox
frontend/src/components/layout/app-shell.tsx                  # Alert store lama dilepas dari topbar
frontend/src/components/layout/sidebar.tsx                    # Versi 0.9.0
frontend/package.json                                         # 0.9.0
```

---

## 3. Perilaku Backend

### 3.1 Tracking publik & privasi

- `GET /api/v1/public/track?ticket=...&phone_last4=...` mencocokkan nomor tiket
  (case-insensitive) dengan `RIGHT(customers.phone_normalized, 4)`.
- `GET /api/v1/public/track/:token` menerima token 32 karakter hex dan hanya
  berlaku bila `public_token_expires_at > now()`.
- `GET /api/v1/public/store?token=...` mengembalikan identitas toko untuk token
  yang valid.
- Ketiga endpoint hanya aktif bila `business_settings.public_tracking_enabled`
  bernilai true.
- Payload publik adalah whitelist eksplisit: nomor tiket, nama/logo toko,
  status + label, waktu diterima, estimasi selesai, perangkat, keluhan,
  diagnosis, status invoice/pembayaran, status pickup, dan ringkasan garansi.
  Nama/telepon/alamat pelanggan, IMEI, kredensial, catatan internal, foto,
  harga modal, dan nama teknisi **tidak pernah** dikirim.
- Kombinasi tiket + 4 digit yang salah mengembalikan `404 NOT_FOUND` dengan
  pesan generik yang tidak mengonfirmasi keberadaan tiket.
- Endpoint publik dilindungi rate limiter **10 request/menit/IP** dan
  mengembalikan `429` + header `Retry-After`.
- Token publik dibuat dari 16 byte acak (`crypto/rand`) sehingga menjadi 32
  karakter hex. Token awal berlaku 30 hari; setelah pickup berlaku 90 hari.
  `GET /service-orders/:id/public-link` mengembalikan token aktif atau
  meregenerasi token yang sudah kedaluwarsa (`regenerated: true`).

### 3.2 QR pada label servis

- Modal cetak label memanggil `GET /service-orders/:id/public-link` saat dibuka
  dan memakai `tracking_url` asli pada QR.
- Bila API gagal (mis. offline), QR memakai fallback
  `${origin}/track?ticket=<ticket_no>` sehingga pelanggan tetap dapat mencari
  tiket dan mengisi 4 digit terakhir HP.
- Halaman `/track` membaca query `?ticket=` dan mengisi kolom nomor tiket
  otomatis dari hasil pindai QR fallback.

### 3.3 Garansi otomatis

- Saat transisi servis ke `picked_up`, hook `CreateForPickupTx` berjalan dalam
  transaksi yang sama dan membuat tepat satu baris `warranties` per service
  order (`ON CONFLICT (service_order_id) DO NOTHING`).
- Durasi garansi diambil dari `invoice_items` (snapshot), lalu
  `service_order_items`, lalu default `business_settings`
  (`default_service_warranty_days`/`default_part_warranty_days`). Nilai 0
  jatuh ke default 30 hari.
- Tipe garansi: `service` (jasa/custom), `part` (sparepart), atau `both`.
  Tanggal mulai adalah tanggal pickup di zona waktu outlet; tanggal akhir
  `start + days`.
- Nomor garansi memakai format `WR-YYMMDD-SEQ` per outlet per hari.
- `GET /warranties` mendukung `q`, `status`, `outlet_id`,
  `expiring_in_days`, `page`, dan `per_page`. Garansi aktif yang melewati
  tanggal akhir otomatis menjadi `expired` saat daftar/detail dibaca.
- `GET /warranties/:id` menyertakan klaim dan invoice terkait. Semua query
  dibatasi `tenant_id` dan outlet yang dapat diakses aktor.

### 3.4 Klaim & rework

- `POST /warranties/:id/claims` hanya untuk garansi `active` dan belum
  melewati tanggal akhir; selain itu `409 WARRANTY_INACTIVE`. Keluhan wajib,
  maksimal 2000 karakter. Nomor klaim `CLM-YYMMDD-SEQ`.
- Transisi status klaim yang diizinkan:

  | Dari | Ke |
  |---|---|
  | `submitted` | `approved`, `in_progress`, `rejected` |
  | `approved` | `in_progress`, `completed`, `rejected` |
  | `in_progress` | `completed`, `rejected` |
  | `completed`/`rejected` | final |

  Transisi lain → `409 INVALID_CLAIM_TRANSITION`.
- `resolution` wajib diisi saat status menjadi `completed` atau `rejected`;
  `handled_by` dan `resolved_at` diisi otomatis.
- `POST /claims/:id/rework` membutuhkan `service.create` + `warranty.claim`.
  Dalam satu transaksi server membuat service order `order_type = 'rework'`
  dengan `rework_of_service_order_id` dan `warranty_claim_id`, riwayat status
  awal `received`, invoice draft baru, token tracking baru, lalu mengubah klaim
  menjadi `in_progress`. Klaim yang sudah punya rework → `409 REWORK_EXISTS`;
  klaim final → `409 CLAIM_NOT_APPROVED`. Audit `warranty.rework` dicatat.

### 3.5 Notifikasi in-app

- Notifikasi tenant dapat berupa broadcast (`user_id IS NULL`) atau personal.
  Status baca broadcast disimpan di `notification_reads` per pengguna;
  notifikasi personal memakai kolom `is_read`/`read_at`.
- `GET /notifications` mendukung `unread=true`, `page`, `per_page`, dan
  mengembalikan header `X-Unread-Count`.
- `GET /notifications/unread-count`, `PATCH /notifications/:id/read`, dan
  `POST /notifications/read-all` tersedia untuk semua pengguna tenant.
- Hook otomatis:
  - `ready_for_pickup` → notifikasi `service_ready` + pesan WA siap diambil.
  - `picked_up` → notifikasi `service_picked`.
  - klaim baru → notifikasi `warranty_claim` + pesan WA konfirmasi klaim.
- Setiap notifikasi memiliki `dedupe_key` unik per tenant sehingga hook yang
  terpanggil ulang tidak membuat baris ganda.

### 3.6 Outbox WhatsApp

- `whatsapp_messages` menyimpan telepon ternormalisasi (`08xx` → `62xx`),
  template, payload, body final, provider, status, attempts, error, jadwal,
  dan `dedupe_key`.
- Status: `queued` → `processing` → `sent`/`failed`; `cancelled` disediakan
  untuk pembatalan manual di masa depan.
- Adapter provider:
  - `noop` — tidak mengirim, mengembalikan ID `noop-*` (default aman).
  - `fonnte` — `POST https://api.fonnte.com/send` dengan token
    `WA_FONNTE_TOKEN`.
  - `wablas` — `POST {WA_WABLAS_URL}/api/send-message` dengan token dan
    opsional secret `WA_WABLAS_SECRET`.
- Provider aktif dibaca dari `business_settings.wa_provider` per tenant/outlet.
- Template bawaan: `status_update`, `ready_pickup`, `invoice`, `reminder`,
  `warranty`, `claim_received`. Tenant dapat menimpa lewat `wa_templates`.
- Worker mengklaim baris dengan `FOR UPDATE SKIP LOCKED`, sehingga aman
  dijalankan lebih dari satu proses. Percobaan gagal dijadwalkan ulang dengan
  backoff (2 dan 4 menit) sampai 3 percobaan, lalu berstatus `failed`.
- `GET /whatsapp-messages` (permission `settings.manage`) mendukung filter
  `status` dan `q` (telepon/tiket/nama pelanggan). Pesan `sent` tidak dapat
  dikirim ulang; `POST /whatsapp-messages/:id/retry` mengembalikan pesan ke
  `queued` dan mereset attempts.

### 3.7 Worker reminder & purge

`go run ./cmd/worker` menjalankan tiga job:

| Job | Interval default | Perilaku |
|---|---|---|
| Outbox | `30s` | Menghabiskan antrean `queued` secara batch |
| Reminder | `15m` | Servis melewati estimasi, perangkat `ready_for_pickup` > 3 hari, garansi berakhir ≤ 7 hari |
| Purge | `1h` | Menghapus kredensial perangkat 30 hari setelah pickup |

- Interval dapat diubah lewat `WORKER_OUTBOX_INTERVAL`,
  `WORKER_REMINDER_INTERVAL`, dan `WORKER_PURGE_INTERVAL`.
- `go run ./cmd/worker -once` menjalankan ketiga job satu kali lalu keluar
  (dipakai pada integration test).
- Notifikasi reminder memakai dedupe key per order/per hari (atau per garansi)
  sehingga worker idempoten dan tidak mengirim pesan ganda.
- Purge mengosongkan `credential_type`, `credential_enc`,
  `credential_key_version`, dan `credential_set_at` ketika
  `credential_purge_at <= now()`.

---

## 4. Frontend

| Route | Halaman | Akses |
|---|---|---|
| `/track` | Cari tiket + 4 digit HP, prefill `?ticket=` | Publik |
| `/track/:token` | Hasil tracking dari QR/tautan | Publik |
| `/warranties` | Tab garansi & klaim, pencarian/filter/pagination | `warranty.view` |
| `/warranties/:id` | Detail garansi, invoice, riwayat klaim, buat klaim, WhatsApp | `warranty.view` |
| `/claims/:id` | Detail klaim, transisi status, rework, WhatsApp | `warranty.view` |
| `/notifications` | Inbox semua/belum dibaca, tandai dibaca | Login |
| `/whatsapp-outbox` | Riwayat outbox, filter status, retry | `settings.manage` |

Perilaku UI:

- Topbar memakai **NotificationMenu** dengan badge unread yang diperbarui
  setiap 60 detik, daftar 6 notifikasi terbaru, tombol tandai semua, dan tautan
  ke inbox lengkap.
- Halaman garansi menampilkan status, tipe, masa berlaku, sisa hari, pelanggan,
  perangkat, servis asal, invoice, syarat, serta riwayat klaim.
- Tombol **Buat Klaim** hanya muncul untuk garansi aktif dan pengguna dengan
  `warranty.claim`.
- Halaman klaim menampilkan progres, keluhan, resolusi, penanganan, dan aksi
  sesuai transisi yang diizinkan. Rework hanya dapat dibuat bila pengguna
  memiliki `warranty.claim` + `service.create`.
- Tombol **Chat Pelanggan** membuka deep link `https://wa.me/<nomor>?text=...`
  dengan normalisasi nomor `08xx` → `62xx`.
- Halaman outbox menampilkan status, provider, attempts, error, jadwal,
  body pesan, tautan tiket, dan tombol kirim ulang untuk pesan `failed`.
- Semua tombol terhubung ke API nyata; tidak ada tombol mati.

---

## 5. Permission & Keamanan

| Permission | Kemampuan |
|---|---|
| `warranty.view` | Daftar/detail garansi dan klaim |
| `warranty.claim` | Membuat klaim, mengubah status, rework |
| `service.create` | Prasyarat tambahan untuk membuat order rework |
| `settings.manage` | Melihat dan mengirim ulang outbox WhatsApp |

- Seed RBAC: Owner dan Admin memiliki seluruh permission tenant. Teknisi dan
  Kasir memiliki `warranty.view` + `warranty.claim`; rework tetap membutuhkan
  `service.create` yang juga dimiliki keduanya.
- Notifikasi in-app tidak memerlukan permission tambahan, tetapi hanya
  menampilkan notifikasi milik pengguna atau broadcast tenant-nya.
- Endpoint publik tidak memakai JWT, tetapi dibatasi field whitelist, token
  acak, expiry, dan rate limit.
- Semua query tenant-scoped memakai `tenant_id`; akses outlet diperiksa
  terhadap `OutletIDs` aktor. Data tenant lain menghasilkan `404`/`403`.
- Token provider WhatsApp hanya dibaca dari environment dan tidak pernah
  ditulis ke log atau respons API.

---

## 6. Pengujian

Integration test baru di `backend/tests/integration/tracking_warranty_test.go`:

- `TestPublicTrackingPrivacyAndTokenExpiry` — public-link, lookup tiket + 4
  digit, whitelist field (tidak ada nama/telepon/alamat/IMEI/PIN/catatan
  internal/harga modal), token valid, digit salah generik, token kedaluwarsa,
  dan regenerasi token.
- `TestPublicTrackingRateLimit` — 10 request lolos, request ke-11 `429` dengan
  `Retry-After`.
- `TestWarrantyClaimReworkAndNotifications` — garansi otomatis 45 hari, klaim,
  notifikasi in-app + outbox, transisi klaim, rework order tertaut + invoice,
  dan isolasi tenant.
- `TestWorkerRemindersOutboxAndCredentialPurge` — reminder idempoten, outbox
  `noop` terkirim, dan purge kredensial setelah 30 hari.
- `TestMigrationUpDownUp` — migrasi 17 naik/turun/naik bersih.

Hasil:

```bash
cd backend
go build ./...
go vet ./...
go test ./... -count=1
# PASS — seluruh paket, termasuk integration suite

go test ./tests/integration -run "TestMigrationUpDownUp|TestPublicTracking|TestWarrantyClaim|TestWorkerReminders" -count=1 -v
# PASS — 5 test Tahap 9

cd frontend
npm run typecheck
npm run build
# PASS — hanya peringatan ukuran chunk > 500 kB
```

`docs/openapi.yaml` diperbarui menjadi versi **0.9.0** dengan endpoint tracking
publik, garansi/klaim, notifikasi, outbox WhatsApp, dan public-link.

---

## 7. Batasan & Langkah Berikutnya

Batasan yang disengaja pada tahap ini:

- Belum ada halaman pengaturan UI untuk memilih provider WhatsApp atau
  mengubah template; nilai dibaca dari `business_settings` (UI menyusul pada
  Tahap 10).
- Notifikasi real-time memakai polling 60 detik; belum ada WebSocket/SSE.
- Outbox belum menyediakan pembatalan manual, jadwal kirim dari UI, atau
  lampiran media.
- Reminder berjalan dari proses worker terpisah; deployment production perlu
  menjalankan `cmd/worker` sebagai service/container sendiri.
- Tracking publik menampilkan diagnosis dan keluhan apa adanya; redaksi
  tambahan dapat ditambahkan bila diperlukan.
- Uji beban konkurensi worker dan E2E Playwright dijadwalkan Tahap 11.

Tahap berikutnya adalah **Tahap 10 — Laporan, Pengaturan & Dashboard Bisnis**:
laporan penjualan/laba, pengaturan toko dan template WhatsApp, manajemen
pengguna/outlet, serta audit log.
