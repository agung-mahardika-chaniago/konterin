# 08 — UI/UX Guideline

**Versi:** 1.0 (Tahap 1) · **Karakter:** SaaS profesional, bersih, cepat, ramah teknisi

---

## 1. Prinsip Desain

1. **Cepat di meja teknisi** — aksi utama maksimal 2 klik dari daftar servis.
2. **Jelas, bukan ramai** — satu fokus per layar; informasi sekunder dilipat.
3. **Konsisten** — status, istilah, dan posisi tombol sama di seluruh aplikasi.
4. **Tahan salah input** — konfirmasi untuk aksi destruktif, validasi inline.
5. **Bisa dipakai dari HP** — target sentuh besar, form satu kolom, tabel jadi kartu.
6. **Terasa hidup** — skeleton, toast, progress, bukan layar kosong tanpa penjelasan.
7. **Bahasa manusia** — hindari istilah teknis; pesan error menjelaskan solusi.

---

## 2. Design Token

### 2.1 Warna

| Token | Hex | Pemakaian |
|---|---|---|
| `brand-950` | `#071A33` | Sidebar, teks judul |
| `brand-900` | `#0B2447` | Tombol primer, header |
| `brand-800` | `#10325F` | Hover primer |
| `brand-700` | `#164278` | Aksen sidebar aktif |
| `brand-600` | `#1D5AA6` | Link, ikon aktif |
| `brand-100` | `#DBEAFE` | Background chip biru |
| `brand-50` | `#EFF6FF` | Background hover |
| `accent-500` | `#06B6D4` | Aksen, grafik, focus ring |
| `accent-600` | `#0891B2` | Hover aksen |
| `accent-50` | `#ECFEFF` | Highlight ringan |
| `surface` | `#FFFFFF` | Card, modal |
| `background` | `#F6F8FB` | Latar halaman |
| `border` | `#E2E8F0` | Garis, pemisah |
| `text` | `#0F172A` | Teks utama |
| `muted` | `#64748B` | Teks sekunder |
| `success` | `#10B981` | Lunas, selesai |
| `warning` | `#F59E0B` | Menunggu, stok menipis |
| `danger` | `#EF4444` | Batal, error, terlambat |
| `info` | `#3B82F6` | Info, dikerjakan |
| `violet` | `#8B5CF6` | Menunggu sparepart |

Rasio kontras teks utama terhadap surface ≥ 12:1; teks muted ≥ 4.5:1 pada
ukuran 14 px. Warna tidak menjadi satu-satunya penanda status — selalu ada
label teks dan/atau ikon.

### 2.2 Tipografi

| Elemen | Font | Ukuran / Line-height | Weight |
|---|---|---|---|
| H1 halaman | Plus Jakarta Sans | 24/32 | 700 |
| H2 section | Plus Jakarta Sans | 18/26 | 600 |
| Body | Plus Jakarta Sans | 14/20 | 400 |
| Label form | Plus Jakarta Sans | 13/18 | 500 |
| Caption/meta | Plus Jakarta Sans | 12/16 | 400 |
| Angka statistik | Plus Jakarta Sans | 28/34 | 700 (tabular-nums) |
| Kode/tiket | JetBrains Mono | 13/20 | 500 |

### 2.3 Spacing, radius, shadow

- Skala spacing: 4, 8, 12, 16, 20, 24, 32, 40, 48 px.
- Radius: `8px` input/tombol, `12px` card, `16px` modal, `999px` badge.
- Shadow card: `0 1px 2px rgba(15,23,42,.04), 0 1px 3px rgba(15,23,42,.06)`.
- Shadow modal: `0 24px 48px rgba(15,23,42,.18)`.
- Border: 1px `#E2E8F0`; fokus: ring 2px `accent-500` + offset 2px.

### 2.4 Ikon

- Lucide React, ukuran 16/18/20/24, stroke 1.75.
- Ikon selalu berpasangan dengan label pada aksi utama; icon-only hanya untuk
  aksi sekunder dengan `aria-label` + tooltip.

---

## 3. Layout

### 3.1 App shell desktop (≥ 1024 px)

```
┌──────────────┬───────────────────────────────────────────────────────────┐
│  FixPhone    │  Outlet: Pusat ▾      🔍 Cari tiket...      🔔  Budi ▾   │ 64px
│  ─────────   ├───────────────────────────────────────────────────────────┤
│  Dashboard   │                                                           │
│  Servis      │   H1 + breadcrumb                        [Aksi utama]     │
│  Pelanggan   │                                                           │
│  ─────────   │   ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐        │
│  Sparepart   │   │ Stat 1  │ │ Stat 2  │ │ Stat 3  │ │ Stat 4  │        │
│  Inventori   │   └─────────┘ └─────────┘ └─────────┘ └─────────┘        │
│  ─────────   │                                                           │
│  Kasir       │   ┌───────────────────────────┐ ┌─────────────────────┐   │
│  Invoice     │   │ Grafik pendapatan         │ │ Perlu tindakan      │   │
│  Laporan     │   └───────────────────────────┘ └─────────────────────┘   │
│  ─────────   │                                                           │
│  Garansi     │   ┌───────────────────────────────────────────────────┐   │
│  Pengaturan  │   │ Tabel servis terbaru                              │   │
│              │   └───────────────────────────────────────────────────┘   │
│  ─────────   │                                                           │
│  Bantuan     │                                                           │
└──────────────┴───────────────────────────────────────────────────────────┘
   264px                         padding 24–32px, max-width 1440px
```

- Sidebar navy, item aktif berlatar `brand-700` + indikator cyan 3 px.
- Sidebar dapat di-collapse menjadi 76 px (ikon + tooltip); state disimpan.
- Topbar: outlet switcher (jika > 1), pencarian global (tiket/nama/telepon),
  notifikasi, menu profil.
- Konten memakai `max-width: 1440px`, padding 24 px (desktop) / 16 px (mobile).

### 3.2 Tablet (768–1023 px)

- Sidebar collapse otomatis menjadi ikon atau drawer.
- Grid statistik 2 kolom; grafik full width.
- Tabel tetap tabel dengan horizontal scroll; kolom sekunder disembunyikan.

### 3.3 Mobile (< 768 px)

- Topbar ringkas: menu ☰, judul, notifikasi, profil.
- Navigasi utama drawer dari kiri.
- Kartu statistik horizontal scroll-snap (2 terlihat).
- Tabel → daftar kartu: tiket + badge status + perangkat + pelanggan + waktu.
- Form satu kolom; tombol simpan **sticky** di bawah.
- Aksi cepat melayang (FAB) "Servis Baru" di halaman servis/dashboard.
- Target sentuh minimum 44×44 px.

---

## 4. Navigasi Sidebar

| Grup | Item | Route | Permission |
|---|---|---|---|
| Utama | Dashboard | `/dashboard` | `dashboard.view` |
| Operasional | Servis | `/services` | `service.view` |
| | Pelanggan | `/customers` | `customer.view` |
| Inventori | Sparepart | `/spareparts` | `sparepart.view` |
| | Pergerakan Stok | `/inventory/movements` | `inventory.view` |
| | Pembelian | `/inventory/purchases` | `inventory.purchase` |
| Keuangan | Kasir | `/cashier` | `payment.create` |
| | Invoice | `/invoices` | `invoice.view` |
| | Pengeluaran | `/expenses` | `expense.view` |
| | Laporan | `/reports` | `report.view` |
| Purna Jual | Garansi | `/warranties` | `warranty.view` |
| | Klaim | `/claims` | `warranty.view` |
| Pengaturan | Pengguna | `/settings/users` | `user.view` |
| | Outlet | `/settings/outlets` | `outlet.view` |
| | Toko | `/settings/store` | `settings.view` |
| | Audit Log | `/settings/audit` | `audit.view` |

Item tanpa permission tidak dirender. Route tetap dilindungi server.

---

## 5. Status Badge

| Status | Label | Warna | Ikon |
|---|---|---|---|
| `received` | Diterima | Slate | inbox |
| `waiting_diagnosis` | Menunggu Diagnosis | Amber | search |
| `waiting_approval` | Menunggu Persetujuan | Orange | clock |
| `waiting_part` | Menunggu Sparepart | Violet | package |
| `in_progress` | Sedang Diperbaiki | Blue | wrench |
| `testing` | Testing | Cyan | flask |
| `completed` | Selesai | Emerald | check-circle |
| `ready_for_pickup` | Siap Diambil | Teal | bell |
| `picked_up` | Sudah Diambil | Green (solid) | check-badge |
| `cancelled` | Batal | Red | x-circle |

Badge: latar `color-50`, teks `color-700`, border `color-200`, titik/ikon 8 px,
radius penuh, tinggi 24 px. Tidak boleh hanya warna.

Status pembayaran: `unpaid` (Belum Lunas, red), `partial` (DP/Sebagian, amber),
`paid` (Lunas, emerald), `void` (Void, slate).
Prioritas: `normal` (slate), `express` (amber + ikon zap).

---

## 6. Komponen Inti

| Komponen | Perilaku |
|---|---|
| `PageHeader` | Judul, deskripsi singkat, breadcrumb, slot aksi |
| `StatCard` | Ikon, label, angka besar, delta kecil, klik menuju daftar terfilter |
| `DataTable` | Search, filter, sorting, pagination, kolom aksi, state kosong/loading |
| `StatusBadge` | Mapping status → label/warna/ikon |
| `StatusStepper` | Alur status horizontal (desktop) / vertikal (mobile) |
| `ServiceForm` | Section: Pelanggan, Perangkat, Keluhan, Estimasi, Teknisi, Catatan |
| `CustomerPicker` | Cari/daftar pelanggan + tombol "Pelanggan Baru" |
| `ItemsEditor` | Baris jasa/sparepart, subtotal live (tampilan), validasi stok |
| `PhotoUploader` | Drag & drop/kamera, preview, hapus, progress, batas 5 file |
| `Timeline` | Riwayat status kronologis dengan pelaku & catatan |
| `CurrencyInput` | Format ribuan otomatis, nilai integer |
| `PhoneInput` | Prefix +62, normalisasi |
| `ConfirmDialog` | Aksi destruktif dengan konfirmasi teks/alasan |
| `EmptyState` | Ikon, penjelasan, CTA relevan |
| `LoadingSkeleton` | Skeleton sesuai bentuk konten (bukan spinner penuh layar) |
| `ErrorState` | Pesan manusiawi + tombol "Coba Lagi" + kode error |
| `Toast` | Sukses/gagal, 4 detik, dapat ditutup |
| `PermissionGuard` | Render anak hanya jika permission ada |
| `OutletSwitcher` | Hanya muncul bila pengguna punya >1 outlet |

---

## 7. Daftar Halaman

### Publik
- `/login`
- `/track` — form tiket + 4 digit HP
- `/track/:token` — hasil tracking
- `/print/invoice/:id` — A4
- `/print/receipt/:id?width=58|80` — struk
- `/print/label/:id?width=58|80` — label
- `/404`, `/403`

### Aplikasi
- `/dashboard`
- `/services`, `/services/new`, `/services/:id`
- `/customers`, `/customers/:id`
- `/spareparts`, `/spareparts/new`, `/spareparts/:id`
- `/inventory/movements`, `/inventory/purchases`, `/inventory/adjustments`
- `/cashier`, `/invoices`, `/invoices/:id`
- `/expenses`
- `/reports?tab=revenue|services|profit|technicians|stock`
- `/warranties`, `/warranties/:id`, `/claims`, `/claims/:id`
- `/settings/store`, `/settings/outlets`, `/settings/users`,
  `/settings/audit`, `/settings/whatsapp`, `/settings/backup`

---

## 8. Wireframe Kunci

### 8.1 Dashboard

```
Halo, Budi 👋                                    [ + Servis Baru ]
Ringkasan hari ini · Rabu, 3 November 2026

┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐
│ Servis     │ │ Menunggu   │ │ Dikerjakan │ │ Belum      │
│ hari ini   │ │ pengerjaan │ │            │ │ diambil    │
│    12      │ │     5      │ │     7      │ │     4      │
└────────────┘ └────────────┘ └────────────┘ └────────────┘
┌────────────┐ ┌─────────────────────────────────────────────┐
│ Pendapatan │ │ Pendapatan 30 hari        [7 hari][30 hari] │
│ hari ini   │ │        ╱╲    ╱╲                             │
│ Rp2.350.000│ │   ╱╲  ╱  ╲__╱  ╲__                          │
└────────────┘ └─────────────────────────────────────────────┘
┌──────────────────────────────┐ ┌────────────────────────────┐
│ Perlu tindakan (3)           │ │ Aktivitas teknisi          │
│ ⚠ 2 servis lewat estimasi    │ │ Andi   8 selesai · 1j 20m  │
│ ⏰ 1 HP belum diambil 3 hari │ │ Rudi   5 selesai · 2j 10m  │
└──────────────────────────────┘ └────────────────────────────┘
```

### 8.2 Daftar Servis

```
Servis HP                                     [ + Servis Baru ]
[🔍 Cari tiket / nama / no. HP]  [Status ▾] [Teknisi ▾] [Tanggal ▾]

┌──────────────┬──────────────────┬──────────────┬───────────────┬────────┐
│ TIKET        │ PELANGGAN        │ PERANGKAT    │ STATUS        │ AKSI   │
├──────────────┼──────────────────┼──────────────┼───────────────┼────────┤
│ FP-261103-01 │ Siti · 0812••••  │ Samsung S22  │ ● Sedang      │ ⋯      │
│ 09:12        │                  │ Layar mati   │   Diperbaiki  │        │
├──────────────┼──────────────────┼──────────────┼───────────────┼────────┤
│ FP-261103-02 │ Andi · 0857••••  │ iPhone 13    │ ● Siap        │ ⋯      │
│ 09:40        │                  │ Baterai      │   Diambil     │        │
└──────────────┴──────────────────┴──────────────┴───────────────┴────────┘
                                          1–20 dari 143  ‹ 1 2 3 ›
```

Aksi baris: Lihat detail, Ubah status, Cetak label, Hubungi via WA.
Di mobile, baris berubah menjadi kartu dua baris dengan badge di kanan atas.

### 8.3 Detail Servis

```
← Servis · FP-261103-0001        [Ubah Status ▾] [Cetak Label] [⋯]
Diterima 09:12 · Teknisi Andi · Prioritas Normal

┌──────────────────────────────────┬───────────────────────────────┐
│ [Ringkasan] [Diagnosis] [Biaya]  │  STATUS                       │
│ [Foto] [Riwayat] [Pembayaran]    │  ● Sedang Diperbaiki          │
│                                  │  Estimasi selesai             │
│ Pelanggan                        │  4 Nov 2026, 17:00            │
│ Siti · 0812-3456-7890            │  [Hubungi via WA]             │
│ Samsung Galaxy S22 · Putih       │                               │
│ Keluhan: layar tidak menyala     │  TAGIHAN                      │
│                                  │  Total     Rp850.000          │
│ Diagnosis                        │  Dibayar   Rp300.000          │
│ LCD rusak, perlu penggantian     │  Sisa      Rp550.000          │
│                                  │  [Terima Pembayaran]          │
└──────────────────────────────────┴───────────────────────────────┘
```

- Tab "Biaya" berisi item jasa/sparepart + tombol tambah.
- Tab "Riwayat" menampilkan timeline lengkap.
- Panel kanan sticky di desktop, menjadi card di atas konten pada mobile.

### 8.4 Kasir

```
Kasir                                    Outlet: Pusat ▾
┌────────────────────────────────┬──────────────────────────────┐
│ Cari tiket / scan QR           │  FP-261103-0001              │
│ [..........................]   │  Siti · Samsung S22          │
│                                │  ─────────────────────────   │
│ Hasil:                         │  Jasa ganti LCD   Rp200.000  │
│ FP-261103-0001 · Siti          │  LCD OEM          Rp650.000  │
│ FP-261102-0014 · Budi          │  ─────────────────────────   │
│                                │  Diskon           −Rp50.000  │
│                                │  TOTAL            Rp800.000  │
│                                │  Dibayar          Rp300.000  │
│                                │  Sisa             Rp500.000  │
│                                │  [Tunai] [Transfer] [QRIS]   │
│                                │  Nominal [ 500.000 ]         │
│                                │  [ Terima Pembayaran ]       │
└────────────────────────────────┴──────────────────────────────┘
```

### 8.5 Tracking Publik

```
┌──────────────────────────────────────────┐
│              FixPhone                    │
│      Status Servis HP Anda               │
│                                          │
│  FP-261103-0001                          │
│  ● Sedang Diperbaiki                     │
│  ▓▓▓▓▓▓▓▓▓▓░░░░░░  60%                   │
│                                          │
│  Diterima      3 Nov 2026, 09:12         │
│  Estimasi      4 Nov 2026, 17:00         │
│  Perangkat     Samsung Galaxy S22        │
│  Diagnosis     LCD rusak                 │
│  Pembayaran    DP · Sisa Rp550.000       │
│  Pengambilan   Belum diambil             │
│                                          │
│  Ada pertanyaan? [Chat via WhatsApp]     │
└──────────────────────────────────────────┘
```

Mobile-first, satu kolom, tanpa data internal. Progress bar mengikuti status
(1 dari 10 langkah, dengan pengelompokan visual).

---

## 9. Form & Validasi

- Label di atas field; placeholder hanya contoh, bukan pengganti label.
- Field wajib ditandai `*` dan dijelaskan "Wajib diisi" di atas form.
- Validasi saat blur + saat submit; pesan di bawah field, merah, spesifik.
- Kelompokkan dengan section dan judul; form panjang dipecah visual, bukan
  multi-halaman (kecuali wizard servis jika diperlukan).
- Tombol utama kanan bawah (desktop) / sticky bottom (mobile); tombol batal
  sekunder.
- Simpan menampilkan spinner + disabled untuk mencegah double submit.
- Perubahan belum tersimpan → konfirmasi saat meninggalkan halaman.
- Input uang: format ribuan otomatis, tidak menerima huruf, nilai 0 valid untuk
  estimasi tetapi tidak untuk pembayaran.
- Input telepon: normalisasi ke `62…`, tampilkan format lokal.
- Upload foto: boleh dari kamera HP; tampilkan progress & kompresi.

---

## 10. State & Microcopy

| State | Tampilan |
|---|---|
| Loading daftar | Skeleton baris (5–8) sesuai tabel |
| Loading kartu | Skeleton angka + grafik |
| Empty servis | Ikon tiket, "Belum ada servis hari ini." + tombol "Terima Servis" |
| Empty pencarian | "Tidak ada hasil untuk '…'. Coba kata kunci lain." |
| Error jaringan | "Gagal memuat data. Periksa koneksi lalu coba lagi." + [Coba Lagi] |
| Error validasi | Pesan per field + ringkasan di atas form |
| Sukses simpan | "Servis FP-261103-0001 berhasil dibuat." |
| Sukses bayar | "Pembayaran Rp300.000 diterima. Sisa Rp550.000." |
| Konfirmasi batal | "Batalkan servis? Tindakan ini mengembalikan stok dan tidak dapat dibatalkan." |

Nada bahasa: sopan, singkat, aktif, tanpa menyalahkan pengguna. Hindari
"Error 500", "Null", "Invalid". Contoh: "Data tidak dapat disimpan. Coba lagi
atau hubungi admin (kode: ABC123)."

---

## 11. Cetak

### 11.1 Invoice A4

- Margin 12 mm; header logo kiri (maks 180×60 px), nama & alamat toko;
  kanan: "INVOICE", nomor, tanggal, status.
- Blok "Ditagihkan kepada" + info perangkat/tiket.
- Tabel: deskripsi, qty, harga, diskon, subtotal.
- Ringkasan: subtotal jasa, subtotal sparepart, diskon, total, dibayar, sisa.
- Footer: syarat garansi, rekening/QRIS, tanda tangan, "Terima kasih".
- Nomor halaman bila > 1; hindari baris tabel terpotong (`break-inside: avoid`).

### 11.2 Struk thermal

| Ukuran | Lebar kertas | Area cetak | Font | Karakter/baris |
|---|---|---|---|---|
| 58 mm | 58 mm | 48 mm | 9–10 px monospace | ±32 |
| 80 mm | 80 mm | 72 mm | 10–11 px monospace | ±42 |

- Header toko, nomor tiket/invoice, tanggal, kasir.
- Item dipotong jika terlalu panjang (2 baris maks) + qty × harga.
- Total, dibayar, sisa, metode.
- Footer: garansi + "Barang yang sudah diambil tidak dapat dikembalikan"
  (disesuaikan kebijakan toko).
- `@page { size: 58mm auto; margin: 2mm; }` dan varian 80 mm.
- Tidak ada warna latar; hitam-putih; gunakan `print-color-adjust: exact` hanya
  bila perlu.

### 11.3 Label servis

- Ukuran 58×40 mm (default) / 80×50 mm.
- Isi: nama toko kecil, nomor tiket besar + QR, perangkat, nama pelanggan
  (depan), keluhan singkat, tanggal, "Simpan label ini".
- QR mengarah ke `https://…/track/{token}`.

---

## 12. PWA

- `manifest.webmanifest`: nama "FixPhone", `theme_color` `#0B2447`,
  `background_color` `#F6F8FB`, ikon 192/512 + maskable.
- Service worker: cache app shell & aset statis; API tidak di-cache.
- Banner offline: "Anda sedang offline. Data terbaru mungkin belum tampil."
- Tombol "Install aplikasi" muncul bila `beforeinstallprompt` tersedia.
- Tidak ada antrean mutasi offline di MVP (cegah data konflik).

---

## 13. Aksesibilitas

- Semua input memiliki `<label>` terhubung (`htmlFor`/`id`).
- Fokus terlihat jelas; urutan tab logis; modal memerangkap fokus & Escape.
- Ikon dekoratif `aria-hidden`; ikon aksi punya `aria-label`.
- Tabel memakai `<th scope="col">`; badge status menyertakan teks.
- Kontras teks ≥ 4.5:1; target sentuh ≥ 44 px.
- Tidak ada informasi penting yang hanya disampaikan lewat warna.
- Notifikasi toast diumumkan via `aria-live="polite"`.
- Grafik memiliki ringkasan teks/tabel alternatif.
- Hormati `prefers-reduced-motion`.

---

## 14. Performa UI

- Route-based code splitting; komponen berat (Recharts, scanner) di-lazy-load.
- Gambar thumbnail di daftar; gambar penuh hanya di detail.
- Debounce pencarian 300 ms; minimal 2 karakter.
- Pagination server-side; tidak memuat seluruh data.
- Optimistic update hanya untuk aksi reversibel (mis. tandai notifikasi dibaca).
- Skeleton, bukan layout shift.

---

## 15. Checklist Review UI per Halaman

- [ ] Ada judul & breadcrumb yang jelas.
- [ ] Aksi utama terlihat tanpa scroll.
- [ ] Loading, empty, error, success state lengkap.
- [ ] Responsif 360/768/1280 px.
- [ ] Keyboard & screen reader dasar berfungsi.
- [ ] Tidak ada tombol tanpa aksi.
- [ ] Format Rupiah/tanggal konsisten.
- [ ] Konfirmasi untuk aksi destruktif.
- [ ] Teks bahasa Indonesia, tanpa jargon.
- [ ] Print (bila ada) diuji di ukuran sebenarnya.
