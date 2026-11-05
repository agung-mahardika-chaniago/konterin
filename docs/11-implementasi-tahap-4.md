# 11 — Implementasi Tahap 4: Frontend Layout & Dashboard

**Status:** selesai ✅ **Versi aplikasi:** 0.4.0 ✅ **Tanggal:** 4 November 2026
**Acuan:** [`05 — Roadmap MVP`](05-roadmap-mvp.md) Tahap 4, [`06 — Kontrak API`](06-kontrak-api.md), [`08 — UI/UX Guideline`](08-uiux-guideline.md)

---

## 1. Tujuan Tahap 4

Tahap 4 mengubah aplikasi dari sekadar halaman login menjadi aplikasi kerja:
kerangka navigasi yang nyaman dipakai teknisi di HP maupun owner di desktop,
serta dashboard yang menampilkan angka nyata dari database.

Yang dihasilkan:

1. App shell: sidebar yang dapat diciutkan, topbar dengan outlet switcher dan
   menu profil, drawer navigasi untuk mobile.
2. Design token navy/cyan, komponen UI dasar, dan halaman yang konsisten.
3. Dashboard 7 kartu statistik, grafik Recharts 7/30 hari, panel peringatan,
   dan aktivitas teknisi.
4. Empat endpoint dashboard: `summary`, `charts`, `alerts`, `technicians`.
5. Skeleton, empty state, error state, dan tombol muat ulang.
6. PWA: manifest, ikon, dan service worker app shell.
7. Test integrasi untuk angka dashboard, filter outlet, isolasi data, dan
   validasi rentang tanggal.

---

## 2. File yang Ditambahkan / Diubah

### Backend

```
backend/internal/modules/dashboard/{dto,handler,routes,service}.go
backend/internal/router/router.go            # wiring /dashboard/*
backend/internal/pkg/version/version.go      # 0.4.0
backend/tests/integration/dashboard_test.go  # acceptance test Tahap 4
```

### Frontend

```
frontend/index.html                          # manifest, ikon, meta PWA
frontend/public/manifest.webmanifest
frontend/public/sw.js
frontend/public/icons/*                      # 192, 512, apple-touch

frontend/src/app/router.tsx                  # route /dashboard + AppShell
frontend/src/components/layout/app-shell.tsx
frontend/src/components/layout/sidebar.tsx
frontend/src/components/layout/topbar.tsx
frontend/src/components/layout/outlet-switcher.tsx
frontend/src/components/layout/profile-menu.tsx
frontend/src/components/layout/nav-items.ts

frontend/src/components/ui/{alert,badge,button,card,input,label,skeleton,spinner}.tsx
frontend/src/components/shared/{brand-mark,empty-state}.tsx

frontend/src/features/dashboard/api.ts
frontend/src/features/dashboard/use-dashboard.ts
frontend/src/features/dashboard/dashboard-page.tsx
frontend/src/features/dashboard/{stat-card,revenue-chart,service-chart}.tsx
frontend/src/features/dashboard/{alerts-panel,technician-panel}.tsx
frontend/src/features/dashboard/{format,types}.ts

frontend/src/stores/{alert-store,outlet-store}.ts
frontend/src/lib/outlet.ts
frontend/tailwind.config.js                  # token warna, radius, shadow
```

---

## 3. App Shell & Navigasi

### 3.1 Sidebar

- Lebar **264 px** saat terbuka dan **76 px** saat diciutkan; status ciut
  disimpan di `localStorage` (`fixphone.sidebar_collapsed`) sehingga bertahan
  setelah reload.
- Saat ciut, label menjadi `sr-only` dan setiap tautan memiliki `title` agar
  tetap dapat dipahami.
- Item aktif ditandai latar `brand-700` dan garis aksen cyan di sisi kiri.
- Navigasi dikelompokkan per modul (Utama, Operasional, Inventori, Keuangan,
  Purna Jual, Pengaturan). Item yang belum dibangun (Tahap 5+) tampil
  **nonaktif** dengan label tahap, bukan tautan mati.
- Item difilter dengan `hasPermission` dari sesi. Teknisi tidak melihat menu
  pengguna/outlet yang bukan haknya.
- Di mobile, sidebar yang sama dirender sebagai drawer.

### 3.2 Drawer mobile

- Dibuka dari tombol menu di topbar; memiliki `role="dialog"`,
  `aria-modal="true"`, dan `aria-label`.
- Fokus dipindahkan ke elemen pertama di dalam drawer, `Escape` menutup,
  klik backdrop menutup, dan scroll body dikunci selama terbuka.
- Drawer tertutup otomatis saat route berubah.
- Di dalam drawer tersedia outlet switcher dan indikator versi.

### 3.3 Topbar

- Sticky di atas dengan efek blur; berisi tombol menu (mobile), judul halaman,
  tanggal hari ini dalam bahasa Indonesia, outlet switcher (desktop), lonceng
  peringatan, dan menu profil.
- Lonceng menaut ke `#alerts-panel` pada dashboard dan menampilkan jumlah item
  yang perlu tindakan (maksimal `99+`).
- Menu profil: nama, email, role, tautan ubah password, dan logout nyata.
  Mendukung `Escape`, klik luar, serta navigasi `ArrowUp`/`ArrowDown`/`Home`/
  `End`.
- Ada skip link "Lewati ke konten utama" yang muncul saat fokus keyboard.

### 3.4 Outlet switcher

- Sumber outlet adalah `user.outlets` dari sesi, bukan input bebas.
- Satu outlet → ditampilkan sebagai label statis (tanpa kontrol palsu).
- Lebih dari satu outlet → `select` dengan label aksesibel; perubahan langsung
  disimpan di `localStorage` dan dikirim sebagai header `X-Outlet-Id` oleh
  `api-client`.
- Outlet tersimpan divalidasi ulang terhadap outlet milik pengguna saat sesi
  dimuat, sehingga ID lama atau asing tidak pernah dikirim ke API.
- Tanpa outlet → status "Tanpa outlet" dan dashboard menampilkan empty state
  yang meminta penugasan outlet.

---

## 4. Design Token & Komponen

Token di `tailwind.config.js`:

| Token | Nilai | Pemakaian |
|---|---|---|
| `brand-900` | `#0B2447` | Navigasi, judul, tombol utama |
| `brand-700` | `#164278` | Item aktif, aksen grafik servis |
| `accent-500` | `#06B6D4` | Fokus, garis aktif, grafik pendapatan |
| `background` | `#F6F8FB` | Latar halaman |
| `surface` | `#FFFFFF` | Kartu dan topbar |
| `border` | `#E2E8F0` | Garis pemisah |
| `ink` / `muted` | `#0F172A` / `#64748B` | Teks utama / sekunder |
| `success` / `warning` / `danger` | `#10B981` / `#F59E0B` / `#EF4444` | Badge dan status |
| `rounded-card` / `rounded-control` | `12px` / `8px` | Kartu / kontrol |
| `shadow-card` / `shadow-modal` | halus / dalam | Kartu / drawer & menu |

Font: **Plus Jakarta Sans** untuk UI dan **JetBrains Mono** untuk nomor tiket
serta SKU. Komponen UI dibuat lokal dengan gaya yang kompatibel shadcn/ui
(`button`, `card`, `badge`, `alert`, `input`, `label`, `skeleton`, `spinner`).

Target ukuran sentuh minimal **40–44 px** pada tombol navigasi dan kontrol
utama. Kontras teks mengikuti pasangan `ink`/`muted` di atas `surface` dan
`brand-100` di atas `brand-950`.

---

## 5. Endpoint Dashboard

Semua endpoint berada di `/api/v1`, memerlukan login + permission
`dashboard.view`, dan dibatasi tenant dari sesi.

| Method | Path | Query | Keterangan |
|---|---|---|---|
| GET | `/dashboard/summary` | `date`, `outlet_id` | Angka hari ini + pendapatan |
| GET | `/dashboard/charts` | `from`, `to`, `outlet_id` | Seri harian inklusif |
| GET | `/dashboard/alerts` | `outlet_id` | Daftar tindakan (maks. 20 per jenis) |
| GET | `/dashboard/technicians` | `from`, `to`, `outlet_id` | Aktivitas teknisi |

### 5.1 Summary

```json
{
  "success": true,
  "data": {
    "services_today": 12,
    "waiting_count": 5,
    "in_progress_count": 7,
    "completed_count": 3,
    "not_picked_up_count": 4,
    "revenue_today": 2350000,
    "revenue_month": 41250000,
    "total_customers": 318,
    "alerts": { "overdue": 2, "not_picked_up_3d": 1, "low_stock": 3 }
  }
}
```

Arti setiap angka:

| Field | Sumber |
|---|---|
| `services_today` | `service_orders.received_at` pada hari yang dipilih, status ≠ `cancelled` |
| `waiting_count` | Status `received`, `waiting_diagnosis`, `waiting_approval`, `waiting_part` (kondisi saat ini) |
| `in_progress_count` | Status `in_progress`, `testing` (kondisi saat ini) |
| `completed_count` | `completed_at` pada hari yang dipilih, status selesai/siap diambil/sudah diambil |
| `not_picked_up_count` | Status `completed`/`ready_for_pickup` dengan `picked_up_at` kosong |
| `revenue_today` / `revenue_month` | `SUM(payments.amount)` berstatus `posted` pada hari/bulan berjalan |
| `total_customers` | Pelanggan tenant yang belum dihapus (tingkat tenant, bukan per outlet) |
| `alerts.overdue` | Servis melewati `estimated_done_at`, belum diambil/dibatalkan |
| `alerts.not_picked_up_3d` | Selesai lebih dari 3 hari dan belum diambil |
| `alerts.low_stock` | Sparepart aktif dengan `stock <= min_stock` (inventori tingkat tenant) |

Pendapatan dihitung dari **pembayaran posted**, bukan dari invoice, sehingga
angka sesuai uang yang benar-benar diterima.

### 5.2 Timezone

- `date`, `from`, dan `to` ditafsirkan pada timezone outlet aktif
  (`outlets.timezone`, default `Asia/Jakarta`).
- Bila `outlet_id` tidak dikirim, server memakai outlet aktif dari header
  `X-Outlet-Id`; jika tidak ada, outlet pertama yang dapat diakses.
- Batas hari dihitung `[awal hari, awal hari berikutnya)` pada zona outlet.
- Grafik mengelompokkan `paid_at`/`received_at`/`completed_at` dengan
  `AT TIME ZONE` zona outlet, bukan UTC.
- Timezone outlet tidak valid tidak membuat dashboard gagal; server memakai
  WIB (`UTC+7`) sebagai fallback.

### 5.3 Grafik

- `charts` selalu mengembalikan **setiap hari** dalam rentang, termasuk hari
  tanpa transaksi (nilai `0`), sehingga grafik tidak bolong.
- Default 7 hari terakhir; klien meminta 7 atau 30 hari; batas server 90 hari.
- `technicians` default 30 hari; batas 180 hari.
- `from` setelah `to` atau format tanggal salah → `422 VALIDATION_ERROR`.
- Teknisi hanya muncul bila memiliki minimal satu order; `avg_minutes` dihitung
  dari `started_at` (atau `received_at` bila kosong) ke `completed_at`.

### 5.4 Isolasi outlet

- `outlet_id` query harus termasuk outlet yang boleh diakses pengguna; bila
  tidak → `403 OUTLET_FORBIDDEN`.
- Owner/admin memiliki akses semua outlet tenant; teknisi/kasir hanya outlet
  yang ditugaskan.
- ID outlet tenant lain tidak dapat dipakai meskipun UUID-nya diketahui.
- Seluruh agregat memakai `tenant_id` dari sesi; tidak ada `tenant_id` dari
  input klien.

---

## 6. Frontend Dashboard

### 6.1 Data & status

`useDashboardData` memuat empat endpoint secara paralel untuk outlet dan
rentang terpilih. Perilaku:

- Request lama diabaikan bila outlet/rentang berubah (nomor request naik).
- Loading awal → skeleton; muat ulang → tombol berputar tanpa mengosongkan
  data yang sudah tampil.
- Error saat data lama masih ada → data dipertahankan dan banner error muncul.
- Error tanpa data → empty state "Dashboard gagal dimuat" + tombol coba lagi.
- Jumlah badge peringatan dihitung dari panjang tiga daftar alert, dibagi ke
  topbar lewat `alert-store`.

### 6.2 Kartu statistik

Tujuh kartu: servis hari ini, menunggu pengerjaan, dikerjakan, selesai hari
ini, belum diambil, pendapatan hari ini, dan pendapatan bulan ini. Nominal
uang diformat `Intl.NumberFormat` `id-ID` tanpa desimal, dengan ringkasan
kompak (`jt`, `rb`) pada keterangan.

### 6.3 Grafik

- **Pendapatan**: area chart cyan dengan grid horizontal, tooltip rupiah, dan
  tabel `sr-only` berisi nilai per hari untuk pembaca layar.
- **Servis**: bar chart dua seri (masuk navy, selesai cyan) dengan tabel
  `sr-only`.
- Keduanya menampilkan pesan "belum ada data" di tengah grafik ketika rentang
  tidak memiliki nilai.
- Tombol 7/30 hari memakai `aria-pressed` dan rentang inklusif berakhir hari ini
  pada kalender browser.

### 6.4 Panel tindakan & teknisi

- **Perlu tindakan** menggabungkan tiga bagian: lewat estimasi, belum diambil,
  dan stok menipis. Tiap baris memuat nomor tiket, umur hari, status, nama
  pelanggan, dan perangkat/SKU. Ada empty state positif bila tidak ada alert.
- **Aktivitas teknisi** menampilkan inisial, jumlah selesai, jumlah aktif, dan
  rata-rata durasi pengerjaan.

### 6.5 Aksesibilitas

- Skip link, `main` dengan `tabIndex={-1}`, dan fokus yang terlihat pada
  kontrol (`focus-visible` ring cyan).
- Drawer dan menu profil dikelola fokus serta dapat ditutup dengan `Escape`.
- Grafik memiliki `role="img"`, label, dan tabel alternatif.
- Status tidak disampaikan lewat warna saja: badge memuat teks status/umur.

---

## 7. PWA

- `manifest.webmanifest`: nama, `short_name`, `start_url="/"`,
  `display=standalone`, `theme_color=#0B2447`, ikon 192/512 (satu maskable),
  serta kategori `business` dan `productivity`.
- `index.html` memuat manifest, ikon 192, apple-touch-icon, dan meta
  `mobile-web-app-capable`.
- `sw.js` hanya aktif pada build produksi (`import.meta.env.PROD`) dan:
  - mencache app shell (`/`, manifest, ikon),
  - menyajikan navigasi dari cache saat offline,
  - **tidak pernah** mencache respons `/api`,
  - menghapus cache versi lama saat aktivasi.
- Ikon: 192×192, 512×512, dan apple-touch 180×180.

---

## 8. Hasil Pengujian

Lingkungan: Go 1.24.5, PostgreSQL 16.15 portable, Node.js 26.12, Vite 5.4.11.

```text
go build ./...                            OK
go vet ./...                              OK
go test ./... -count=1                    OK (~71s)
  internal/config                          OK
  internal/database                        OK
  tests/integration                        OK
npm run typecheck                         OK
npm run build                             OK
  dist/assets/index-*.js   660.51 kB │ gzip 190.17 kB
  dist/assets/index-*.css   27.37 kB │ gzip   5.73 kB
```

Catatan: Vite memberi peringatan ukuran chunk > 500 kB. Ini bukan kegagalan
build; code-splitting dapat ditambahkan pada Tahap 11/12 bila diperlukan.

Test integrasi Tahap 4 (`backend/tests/integration/dashboard_test.go`):

| Test | Cakupan |
|---|---|
| `TestDashboardSummaryChartsAlertsAndTechnicians` | Angka summary dari data DB; pendapatan dari pembayaran `posted`; 3 jenis alert; grafik 7 dan 30 hari inklusif; aktivitas teknisi (1 selesai, 1 aktif, rata-rata 90 menit); teknisi boleh membuka dashboard; outlet asing → 403; tanggal/rentang invalid → 422 |
| `TestDashboardOutletFilterIsolatesData` | Tanpa `outlet_id` memakai outlet aktif; filter query memisahkan data dua outlet; header `X-Outlet-Id` setara query; jumlah pelanggan tetap tingkat tenant |

Verifikasi manual yang sudah dijalankan pada sesi ini (headless Chrome 156 +
CDP terhadap `npm run dev` + API lokal, akun owner, DB dev):

- Login via form lalu dikirim ke `/dashboard`; angka kartu, grafik, panel
  tindakan, dan aktivitas teknisi tampil tanpa error.
- Multi-outlet dites dengan outlet kedua "Cabang Demo": topbar memunculkan
  `<select>` dua pilihan; memilih "Outlet Demo (MAIN)" membuat permintaan
  `summary` berikutnya membawa `X-Outlet-Id: <id-MAIN>` (tertangkap via CDP
  `Network.requestWillBeSent`); label header halaman ikut berubah.
- Pilihan outlet bertahan setelah reload halaman (`select.value` dan tombol
  `fixphone.active_outlet` di `localStorage` — bukan token).
- Navigasi keyboard: menu profil dibuka dengan `Enter`, fokus otomatis ke item
  pertama (`role="menuitem"`), `Escape` menutupnya.
- Breakpoint 768 px (versi sidebar ikon), 1280+ xl grid, dan 390×844: drawer
  mobile terbuka dari tombol topbar, berisi kontrol outlet, dan tertutup via
  `Escape`.
- Toggle 7/30 hari bekerja: tabel pembaca layar berubah dari 7 ke 30 baris.

Tidak ada `error`/`warning` console pada alur di atas (`problems: []`).
Catatan: satu permintaan `401` dari `/auth/refresh` pada kunjungan pertama
sebelum perbaikan session-hint; setelah perbaikan, profil headless baru tidak
lagi memunculkannya. Headless Chrome tidak menggantikan audit aksesibilitas
otsomatic; audit penuh tetap dijadwalkan Tahap 11.

---

## 9. Keamanan & Batasan

- Dashboard hanya untuk pengguna tenant; super admin tanpa tenant → `403`.
- Permission `dashboard.view` diperiksa server; menu frontend hanya lapisan UX.
- `outlet_id` divalidasi terhadap scope pengguna, bukan dipercaya.
- Agregat tidak pernah mengembalikan data lintas tenant.
- Sparepart dan pelanggan bersifat tingkat tenant pada skema saat ini, sehingga
  hitungannya tidak berubah saat outlet diganti. Ini konsisten dengan desain
  database, bukan kebocoran antar-outlet.
- Endpoint bersifat read-only; tidak ada mutasi dari dashboard.
- Rate limit dashboard belum spesifik; dilindungi rate limit global yang ada.

---

## 10. Yang Sengaja Ditunda

| Item | Tahap |
|---|---|
| Manajemen pengguna di UI | 10 |
| Filter tanggal kustom pada dashboard | 10 |
| Top sparepart, tren laba, dashboard bisnis final | 10 |
| Code-splitting bundle & tuning performa | 11–12 |
| E2E Playwright & audit aksesibilitas | 5+ / 11 |
| Install prompt PWA kustom | 9 (opsional) |

---

## 11. Definition of Done Tahap 4

- [x] Dashboard menampilkan angka dari database (diuji integrasi, bukan
      hardcoded).
- [x] Filter outlet bekerja dan terisolasi (diuji integrasi).
- [x] Grafik akurat untuk rentang 7 dan 30 hari (keduanya diuji eksplisit).
- [x] Endpoint `/dashboard/*` memiliki permission, validasi, dan tenant scope.
- [x] Skeleton, empty state, error state, dan muat ulang tersedia.
- [x] PWA manifest, ikon, dan service worker tersedia.
- [x] `go build`, `go vet`, `go test`, `npm run typecheck`, `npm run build`
      lulus.
- [x] Responsif & navigasi keyboard diverifikasi di headless Chrome
      (breakpoint 390/768/1280, drawer `Escape`, menu profil keyboard).
- [x] Tidak ada warning console di headless Chrome (setelah perbaikan meta PWA
      + future flags React Router + session hint bootstrap).
