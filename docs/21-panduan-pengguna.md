# 21 — Panduan Pengguna FixPhone

**Versi:** 1.0.0 · **Terakhir diperbarui:** 14 November 2026
**Untuk:** Owner/Admin, Teknisi, Kasir
**Dokumen terkait:** [`20 — Runbook Operasional`](20-runbook-operasional.md), [`08 — UI/UX Guideline`](08-uiux-guideline.md)

Panduan ini menjelaskan cara memakai FixPhone sehari-hari: menerima servis,
mengerjakan perangkat, membuat invoice, menerima pembayaran, mencetak, dan
menutup kasir. Semua data yang terlihat hanya milik toko dan outlet Anda.

---

## 1. Peran & Hak Akses

| Fitur | Owner | Admin | Teknisi | Kasir |
|---|---|---|---|---|
| Dashboard | ✅ | ✅ | ✅ | ✅ |
| Melihat servis | ✅ | ✅ | ✅ | ✅ |
| Menerima servis baru | ✅ | ✅ | ✅ | ✅ |
| Mengubah status servis | ✅ | ✅ | ✅ | hanya siap diambil / sudah diambil |
| Membatalkan servis | ✅ | ✅ | ❌ | ❌ |
| Menugaskan teknisi | ✅ | ✅ | ❌ | ❌ |
| Melihat kredensial perangkat | ✅ | ❌ | ✅ | ❌ |
| Override pickup belum lunas | ✅ | ❌ | ❌ | ❌ |
| Pelanggan | ✅ | ✅ | lihat saja | tambah/ubah |
| Sparepart & stok | ✅ | ✅ | lihat saja | lihat saja |
| Pembelian & opname | ✅ | ✅ | ❌ | ❌ |
| Invoice | ✅ | ✅ | ❌ | buat/ubah |
| Pembayaran | ✅ | ✅ | ❌ | terima |
| Void invoice/pembayaran | ✅ | ✅ | ❌ | ❌ |
| Pengeluaran | ✅ | ✅ | ❌ | ❌ |
| Laporan | ✅ | ✅ | ❌ | lihat (tanpa laba) |
| Laporan laba & nilai inventori | ✅ | ❌ | ❌ | ❌ |
| Garansi & klaim | ✅ | ✅ | ✅ | ✅ |
| Pengguna & outlet | ✅ | ✅ | lihat outlet | lihat outlet |
| Pengaturan toko | ✅ | ✅ | ❌ | ❌ |
| Outbox WhatsApp | ✅ | ✅ | ❌ | ❌ |
| Audit log | ✅ | ❌ | ❌ | ❌ |
| Backup manual tenant | ✅ | ✅ | ❌ | ❌ |

Tombol yang tidak Anda miliki izinnya tidak ditampilkan. Bila muncul halaman
**403**, berarti akun Anda tidak memiliki izin untuk tindakan tersebut.

### 1.1 Akun Super Admin

Super Admin adalah akun platform (bukan milik toko) untuk operasional FixPhone.
Akun ini tidak melihat data operasional tenant. Pembuatan tenant baru dilakukan
oleh tim operasional melalui CLI server (lihat [`20 — Runbook`](20-runbook-operasional.md) §3.1).

---

## 2. Masuk, Keluar & Password

### 2.1 Masuk

1. Buka `https://<domain-toko>`.
2. Isi **Email** dan **Password**.
3. Klik **Masuk**.

Setelah login, Anda diarahkan ke halaman sesuai peran:

- Owner/Admin → Dashboard
- Teknisi → Dashboard (fokus tugas teknisi)
- Kasir → Dashboard/Kasir

### 2.2 Sesi

- Sesi diperpanjang otomatis saat aplikasi dipakai.
- Menutup browser tidak langsung mengakhiri sesi; gunakan **Keluar** pada menu
  profil bila memakai komputer bersama.
- Jangan menyimpan password di browser komputer bersama.

### 2.3 Ganti password

1. Klik nama Anda di kanan atas → **Ganti Password**.
2. Isi password lama, password baru, dan konfirmasi.
3. Password minimal 8 karakter dan mengandung huruf serta angka.

Bila akun baru dibuat atau password direset admin, Anda wajib mengganti
password saat login pertama.

### 2.4 Lupa password

Tidak ada reset mandiri lewat email. Hubungi **Owner atau Admin** toko untuk
melakukan reset dari halaman **Pengguna → Reset kata sandi**. Setelah direset,
login dengan password sementara lalu segera ganti.

### 2.5 Keluar

Menu profil → **Keluar**. Setelah keluar, refresh token dicabut sehingga sesi
tidak dapat dilanjutkan.

---

## 3. Navigasi & Outlet

Sidebar berisi kelompok menu:

- **Utama** — Dashboard
- **Operasional** — Servis, Pelanggan
- **Inventori** — Sparepart, Pergerakan Stok, Pembelian
- **Keuangan** — Kasir, Invoice, Pengeluaran, Laporan
- **Purna Jual** — Garansi, Notifikasi, Outbox WA
- **Pengaturan** — Pengguna, Outlet, Toko, Audit Log

Di topbar:

- **Switcher outlet** — bila Anda bertugas di lebih dari satu outlet. Data yang
  tampil mengikuti outlet aktif.
- **Lonceng notifikasi** — servis siap diambil, klaim garansi, pengingat.
- **Menu profil** — ganti password dan keluar.

> Data antar-outlet terpisah. Pastikan outlet yang aktif sudah benar sebelum
> menerima servis atau menerima pembayaran.

---

## 4. SOP Harian

### 4.1 Buka toko (Admin/Kasir)

1. Login dan periksa outlet aktif.
2. Buka **Dashboard**: cek servis masuk, menunggu pengerjaan, selesai hari ini,
   dan perangkat belum diambil.
3. Buka **Notifikasi** dan **Outbox WA**: kirim ulang pesan yang gagal.
4. Periksa **Sparepart** yang stoknya di bawah minimum.
5. Pastikan kas kasir sebelumnya sudah sesuai (lihat 4.5).

### 4.2 Terima servis (Kasir/Teknisi/Admin)

1. **Servis → Servis Baru**.
2. Cari pelanggan berdasarkan nama/nomor HP.
   - Bila belum ada, isi nama dan nomor HP; pelanggan dibuat otomatis.
3. Isi data perangkat: merek, model, IMEI/serial (opsional), warna.
4. Catat kelengkapan dan kondisi fisik, misalnya lecet, retak, atau tombol
   rusak. Ini melindungi toko dari klaim yang tidak benar.
5. Isi **keluhan pelanggan** sedetail mungkin.
6. Tambahkan foto perangkat bila perlu (maksimal sesuai pengaturan toko).
7. Simpan. Sistem membuat **tiket servis** dan token tracking publik.
8. Cetak **label servis** dan tempelkan pada perangkat.
9. Berikan nomor tiket atau tautan tracking kepada pelanggan.

> Jangan menulis password/PIN perangkat di kolom keluhan. Gunakan fitur
> **Kredensial** yang terenkripsi dan hanya dapat dibuka oleh teknisi/Owner.

### 4.3 Pengerjaan (Teknisi)

1. Buka **Servis**, filter **Ditugaskan ke saya** atau cari tiket.
2. Ubah status ke **Menunggu Diagnosis** saat perangkat mulai diperiksa.
3. Isi **Diagnosis**: hasil pemeriksaan, penyebab, dan rekomendasi.
4. Tentukan item jasa dan sparepart pada tab item:
   - Cari sparepart; stok berkurang saat item ditambahkan.
   - Sesuaikan qty dan harga bila perlu.
5. Bila pelanggan perlu menyetujui biaya, ubah ke **Menunggu Persetujuan** dan
   hubungi pelanggan.
6. Bila sparepart belum ada, ubah ke **Menunggu Sparepart** dan catat kebutuhan.
7. Saat dikerjakan, ubah ke **Sedang Dikerjakan**.
8. Setelah selesai, ubah ke **Pengujian**, lakukan uji fungsi.
9. Bila lulus, ubah ke **Selesai**. Sistem memberi tahu kasir/admin bahwa
   perangkat siap ditagihkan.
10. Setelah invoice lunas, kasir mengubah ke **Siap Diambil** lalu
    **Sudah Diambil** saat pelanggan mengambil.

Status yang tersedia dan urutan normalnya:

```
Diterima → Menunggu Diagnosis → Menunggu Persetujuan / Menunggu Sparepart
        → Sedang Dikerjakan → Pengujian → Selesai → Siap Diambil → Sudah Diambil
```

Transisi yang tidak diizinkan akan ditolak dengan pesan kesalahan. Status
**Dibatalkan** hanya untuk Owner/Admin dan wajib menyertakan alasan.

### 4.4 Kasir & pembayaran

1. Buka **Kasir**.
2. Pilih tanggal dan outlet.
3. Pada daftar **Tagihan Belum Lunas**, buka invoice servis.
4. Periksa item, diskon, dan total bersama pelanggan.
5. Klik **Bayar**, pilih metode (Tunai, Transfer, QRIS, Kartu, Lainnya), isi
   jumlah. Pembayaran DP diperbolehkan; status menjadi **Dibayar Sebagian**.
6. Setelah lunas, status invoice menjadi **Lunas**.
7. Cetak **Invoice A4** atau **Struk thermal 58/80 mm**.
8. Ubah status servis ke **Siap Diambil**, lalu **Sudah Diambil** saat
   perangkat diserahkan.
9. Garansi dibuat otomatis saat pickup sesuai pengaturan toko.

### 4.5 Tutup kasir

1. Buka **Kasir** dan pilih tanggal hari ini.
2. Cocokkan **Total penerimaan**, **Rekap Metode Bayar**, dan uang fisik.
3. Buka **Laporan → Pendapatan** bila perlu mencetak/ekspor rekap.
4. Catat selisih (bila ada) dan laporkan ke Admin/Owner.
5. Pastikan tidak ada pembayaran yang salah metode; void hanya oleh
   Admin/Owner dengan alasan.

### 4.6 Tutup toko (Admin)

1. Pastikan semua perangkat yang sudah lunas berstatus **Sudah Diambil**.
2. Periksa servis **Siap Diambil** yang belum diambil; hubungi pelanggan.
3. Catat pengeluaran harian (transport, konsumsi, operasional) di
   **Pengeluaran**.
4. Periksa notifikasi dan outbox WhatsApp.
5. Kunci komputer/keluar dari aplikasi.

---

## 5. Panduan Admin/Owner

### 5.1 Dashboard

- Kartu ringkasan: servis masuk, menunggu pengerjaan, selesai hari ini,
  belum diambil.
- Grafik 7/30 hari: servis masuk vs selesai, pendapatan.
- Peringatan: servis melewati estimasi, perangkat belum diambil > 3 hari, stok
  minimum.
- Panel aktivitas teknisi: jumlah aktif dan selesai.

Klik kartu/daftar untuk membuka detail. Angka dihitung dari data nyata sesuai
outlet aktif.

### 5.2 Servis

- **Daftar servis**: cari tiket/nama/HP, filter status, teknisi, tanggal.
- **Detail servis**:
  - Ubah data pelanggan/perangkat.
  - Ubah status dengan catatan.
  - Tugaskan teknisi dan prioritas (Normal/Express).
  - Kelola diagnosis, item jasa/sparepart, foto, kredensial.
  - Lihat riwayat status dan audit.
  - Cetak label, buka invoice, salin tautan tracking.
- **Batalkan servis**: wajib alasan; stok sparepart dikembalikan; invoice yang
  belum dibayar otomatis void. Servis yang sudah memiliki pembayaran tidak
  dapat dibatalkan.

### 5.3 Pelanggan

- Cari berdasarkan nama/HP.
- Tambah/ubah pelanggan dan daftar perangkatnya.
- Buka riwayat servis pelanggan dari halaman detail.
- Hapus pelanggan hanya bila tidak memiliki transaksi; gunakan dengan hati-hati.

### 5.4 Inventori

- **Sparepart**: tambah/edit, atur stok minimum, lihat valuasi dan riwayat.
- **Pembelian**: catat pembelian multi-baris; stok dan harga modal diperbarui.
- **Pergerakan Stok**: seluruh mutasi (pembelian, pemakaian servis, retur,
  opname) beserta referensi.
- **Opname**: sesuaikan stok fisik; selisih tercatat sebagai mutasi.
- **Harga modal** hanya terlihat oleh pemilik izin `inventory.cost.view`
  (Owner).

Gunakan pencarian barcode/SKU bila scanner tersedia.

### 5.5 Invoice & pembayaran

- Invoice dibuat dari servis. Status: **Draft → Belum Dibayar → Dibayar
  Sebagian → Lunas**, atau **Dibatalkan**.
- Diskon diberikan sebelum finalisasi; alasan diskon dicatat.
- **Void invoice** hanya bila belum ada pembayaran posted. Pembayaran yang
  sudah posted harus di-void terlebih dahulu.
- **Void pembayaran** memerlukan alasan dan mengembalikan status invoice.
- Cetak invoice A4, struk thermal, dan label dari halaman terkait.

### 5.6 Pengeluaran

- Catat pengeluaran per kategori (Operasional, Gaji, Sewa, Listrik & Internet,
  Pembelian Alat, Lain-lain).
- Kategori dapat ditambah/diubah.
- Pengeluaran memengaruhi laporan laba.

### 5.7 Laporan

Tersedia: Pendapatan, Servis, Merek, Jasa vs Part, Laba, Pengeluaran, Teknisi,
Stok, dan Nilai Inventori. Pilih outlet dan rentang tanggal, lalu **Ekspor**
untuk mengunduh xlsx.

Laba dan nilai inventori hanya untuk Owner.

### 5.8 Garansi & klaim

- Garansi dibuat otomatis saat pickup sesuai durasi di pengaturan.
- **Garansi** menampilkan masa berlaku dan servis asal.
- **Klaim** dibuat dari garansi; pilih keluhan, tindakan, dan hasil
  (diterima/ditolak). Klaim yang diterima dapat membuat order rework yang
  tertaut ke servis asal.
- Semua klaim tercatat di audit log.

### 5.9 Pengguna & outlet

- **Pengguna**: tambah akun, pilih role (Admin, Teknisi, Kasir), atur outlet,
  reset password, nonaktifkan/hapus.
- **Outlet**: tambah/ubah outlet, kode, alamat, telepon, zona waktu.
- Teknisi hanya dapat ditugaskan pada outlet tempat ia terdaftar.
- Jangan memakai satu akun untuk banyak orang; audit log bergantung pada akun
  individu.

### 5.10 Pengaturan toko

Tab:

- **Identitas Toko** — nama, alamat, telepon, logo, zona waktu.
- **Tiket & Invoice** — format nomor, prefix, footer, syarat.
- **Garansi & Struk** — durasi garansi, teks struk, ukuran kertas.
- **WhatsApp** — provider (`noop`/Fonnte/Wablas), token, template pesan.
- **Backup** — backup manual tenant (unduh file) dan riwayat.

Perubahan pengaturan langsung berlaku untuk transaksi berikutnya.

### 5.11 Audit log

Mencatat aksi sensitif: login, perubahan status, diskon, void, hapus data,
perubahan pengguna/pengaturan, override pickup, dan backup. Gunakan filter
modul/tanggal dan cari nama pengguna.

Audit log **tidak boleh dihapus**; ia adalah bukti operasional.

### 5.12 Notifikasi & outbox WhatsApp

- **Notifikasi** — pesan in-app untuk servis siap diambil, klaim, pengingat.
- **Outbox WA** — status pesan: Menunggu, Terkirim, Gagal, Dibatalkan. Gunakan
  **Kirim ulang** untuk pesan gagal.
- Pengiriman nyata memerlukan token provider di pengaturan. Tanpa token,
  provider `noop` hanya mencatat pesan.

### 5.13 Backup manual tenant

**Pengaturan → Backup → Buat Backup** membuat file berisi data tenant (bukan
seluruh database). File dapat diunduh.

> Backup manual tenant adalah pelengkap, **bukan** pengganti backup harian
> server. Pemulihan bencana dilakukan oleh tim operasional sesuai runbook.

---

## 6. Panduan Teknisi

### 6.1 Melihat tugas

- **Dashboard** menampilkan ringkasan dan aktivitas.
- **Servis** → filter teknisi = Anda, atau status **Menunggu Diagnosis**,
  **Sedang Dikerjakan**, **Menunggu Sparepart**.
- Prioritas **Express** ditandai; kerjakan sesuai instruksi Admin.

### 6.2 Mengerjakan servis

1. Buka detail servis dan baca keluhan, kondisi fisik, serta kelengkapan.
2. Buka **Kredensial** bila perlu (password/PIN perangkat). Setiap pembukaan
   tercatat di audit log.
3. Ubah status dan isi **Diagnosis**.
4. Tambahkan jasa/sparepart. Periksa stok; bila habis, ajukan ke Admin untuk
   pembelian dan ubah status ke **Menunggu Sparepart**.
5. Catat pekerjaan di catatan teknisi (terlihat tim internal).
6. Uji fungsi menyeluruh sebelum **Selesai**: layar, sentuh, kamera, sinyal,
   speaker, mikrofon, tombol, charger, dan fungsi yang dikeluhkan.
7. Pastikan perangkat dirakit kembali dengan benar dan tidak ada bagian
   tertinggal.

### 6.3 Yang tidak boleh dilakukan teknisi

- Membatalkan servis.
- Mengubah harga invoice atau menerima pembayaran.
- Menghapus pelanggan/data.
- Mengubah pengguna, outlet, atau pengaturan toko.
- Membagikan kredensial perangkat ke pihak lain.

### 6.4 Serah terima ke kasir

Setelah **Selesai**, beri tahu kasir bahwa perangkat siap ditagihkan. Jangan
menyerahkan perangkat ke pelanggan sebelum pembayaran diselesaikan, kecuali
Owner memberikan override dengan alasan tertulis.

---

## 7. Panduan Kasir

### 7.1 Menerima servis

Ikuti §4.2. Pastikan nomor HP pelanggan benar karena dipakai untuk tracking
dan WhatsApp. Cetak label dan tempel pada perangkat.

### 7.2 Membuat/mengubah invoice

1. Buka servis → **Invoice** → buat invoice dari item servis.
2. Periksa setiap baris: nama, qty, harga, diskon.
3. Diskon memerlukan alasan dan tercatat.
4. Finalisasi invoice bila pelanggan setuju. Invoice draft belum menjadi
   tagihan resmi.

### 7.3 Menerima pembayaran

1. Buka **Kasir** atau invoice.
2. Klik **Bayar**.
3. Pilih metode dan jumlah. Untuk DP, isi jumlah sebagian.
4. Periksa kembali nominal dan metode sebelum menyimpan.
5. Cetak struk. Untuk pembayaran non-tunai, cocokkan bukti transfer/EDC.
6. Status invoice berubah otomatis.

### 7.4 Void

- **Void pembayaran** hanya oleh Admin/Owner, dengan alasan. Kasir harus
  meminta persetujuan.
- **Void invoice** hanya bila belum ada pembayaran posted.

### 7.5 Serah terima perangkat

1. Pastikan invoice **Lunas**.
2. Ubah status ke **Siap Diambil**, lalu **Sudah Diambil** saat pelanggan
   datang.
3. Periksa identitas/nomor tiket pelanggan.
4. Cetak bukti pickup bila diminta.
5. Bila pelanggan mengambil sebelum lunas, hentikan proses dan minta
   Owner/Admin melakukan override dengan alasan.

### 7.6 Tutup kasir

Ikuti §4.5. Laporkan selisih kas hari itu kepada Admin/Owner; jangan menutupi
selisih dengan transaksi baru.

---

## 8. Tracking Publik

Pelanggan dapat melacak tanpa login:

- Buka `https://<domain>/track`.
- Masukkan **nomor tiket** dan **4 digit terakhir nomor HP**.
- Atau buka tautan/scan QR pada label servis (`/track/<token>`).

Yang terlihat: status, perkiraan selesai, riwayat singkat, dan info toko.
Kredensial, catatan internal, dan data pembayaran **tidak** ditampilkan.

Token tracking berlaku 90 hari setelah pickup. Bila pelanggan kehilangan
tautan, mereka tetap dapat memakai tiket + 4 digit HP.

---

## 9. Cetak

| Dokumen | Lokasi | Ukuran |
|---|---|---|
| Label servis | Detail servis | Thermal label / A4 |
| Invoice | Detail invoice | A4 |
| Struk pembayaran | Kasir / invoice | Thermal 58 mm atau 80 mm |
| Laporan | Laporan → Ekspor | xlsx |

Tips:

- Gunakan dialog print browser dan pilih printer yang benar.
- Untuk thermal, pilih ukuran kertas 58/80 mm dan margin minimum.
- Aktifkan "background graphics" bila logo/garis tidak tercetak.
- Uji sekali per printer sebelum jam sibuk.

---

## 10. Aturan Penting

1. **Satu akun untuk satu orang.** Jangan berbagi password; audit log butuh
   identitas yang jelas.
2. **Periksa outlet aktif** sebelum transaksi.
3. **Jangan menulis data sensitif** (password, PIN, data kartu) di catatan
   bebas. Gunakan fitur kredensial.
4. **Jangan menghapus data** untuk "merapikan". Gunakan pembatalan/void dengan
   alasan agar jejak audit tetap ada.
5. **Pembayaran harus melalui aplikasi** agar kas dan laporan cocok.
6. **Perangkat diserahkan setelah lunas**, kecuali override Owner.
7. **Backup bukan tugas harian kasir**, tetapi jangan menghalangi tim
   operasional saat melakukan backup/restore.
8. **Laporkan** error, data hilang, atau akses mencurigakan segera ke Admin.

---

## 11. Troubleshooting Pengguna

| Masalah | Penyebab umum | Solusi |
|---|---|---|
| Tidak bisa login | Password salah, akun nonaktif, terlalu banyak percobaan | Tunggu 15 menit; minta Admin reset/nonaktifkan |
| Halaman 403 | Role tidak memiliki izin | Minta Owner/Admin menyesuaikan role |
| Data toko lain terlihat | Outlet aktif salah / akun salah | Ganti outlet di topbar; laporkan bila tetap |
| Tombol tidak muncul | Izin tidak dimiliki | Lihat matriks §1 |
| Stok tidak berkurang | Item bukan sparepart / belum disimpan | Periksa tipe item dan simpan ulang |
| Tidak bisa ubah status | Transisi tidak valid / izin kurang | Ikuti alur status §4.3 |
| Invoice tidak bisa void | Sudah ada pembayaran | Void pembayaran dulu (Admin/Owner) |
| Pickup ditolak | Invoice belum lunas | Selesaikan pembayaran atau minta override Owner |
| WhatsApp tidak terkirim | Token/provider belum diatur, pesan gagal | Cek Pengaturan → WhatsApp; kirim ulang dari Outbox |
| Cetak terpotong | Ukuran kertas/margin salah | Pilih 58/80 mm, margin minimum, matikan header/footer |
| Aplikasi lambat | Koneksi internet/server | Muat ulang; laporkan bila > 5 menit |
| Sesi berakhir terus | Cookie diblokir/private mode | Izinkan cookie untuk domain toko |

Bila masalah berlanjut, catat: waktu, halaman, tindakan terakhir, pesan error,
dan tangkapan layar. Sertakan nomor tiket/invoice bila ada, lalu hubungi Admin.

---

## 12. Daftar Istilah

| Istilah | Arti |
|---|---|
| **Tiket** | Nomor unik servis, dipakai pelanggan untuk tracking |
| **Intake** | Proses penerimaan perangkat |
| **Diagnosis** | Hasil pemeriksaan dan rencana perbaikan |
| **DP** | Pembayaran sebagian |
| **Void** | Pembatalan invoice/pembayaran dengan jejak audit |
| **Override pickup** | Izin Owner menyerahkan perangkat sebelum lunas, dengan alasan |
| **Rework** | Servis ulang yang tertaut ke klaim garansi |
| **Outbox** | Antrean pesan WhatsApp yang menunggu dikirim worker |
| **Outlet** | Cabang/lokasi toko |
| **Tenant** | Badan usaha pemilik data (satu atau banyak outlet) |

---

## 13. Dukungan

- **Admin/Owner toko** — masalah akun, izin, pengaturan, pembatalan.
- **Tim operasional FixPhone** — gangguan server, backup/restore, keamanan.
  Kontak dan eskalasi ada di [`20 — Runbook Operasional`](20-runbook-operasional.md) §7.3.

Saat melapor, sertakan:

1. Nama dan role akun.
2. Waktu kejadian (zona waktu outlet).
3. Halaman dan nomor tiket/invoice.
4. Langkah yang sudah dicoba.
5. Pesan error/tangkapan layar.
