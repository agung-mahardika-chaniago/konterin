# 12 — Implementasi Tahap 5: Modul Penerimaan Servis, Pelanggan, Foto & Label

**Status:** selesai ✅ **Versi aplikasi:** 0.5.0 ✅ **Tanggal:** 6 November 2026
**Acuan:** [`05 — Roadmap MVP`](05-roadmap-mvp.md) Tahap 5, [`06 — Kontrak API`](06-kontrak-api.md), [`08 — UI/UX Guideline`](08-uiux-guideline.md)

---

## 1. Tujuan Tahap 5

Tahap 5 mengimplementasikan alur inti penerimaan servis smartphone (intake counter)
dari saat pelanggan datang membawa HP rusak sampai nomor tiket terbit dan label
stiker thermal tercetak:

1. **Manajemen Pelanggan**: CRUD pelanggan, pencarian cepat, normalisasi nomor telepon
   ke format `628xxx`, serta pencegahan nomor duplikat per tenant.
2. **Perangkat Pelanggan**: Registrasi merek, model, nomor seri/IMEI, dan warna HP.
3. **Penerimaan Servis Atomik**: Nomor tiket unik dari sequence database (`TKT-...`),
   keluhan kerusakan, checklist kelengkapan (unit, baterai, charger, dll.), dan
   kondisi fisik saat intake.
4. **Kredensial Aman**: Penyimpanan PIN/Password/Pola kunci layar dengan enkripsi
   **AES-256-GCM** (write-only). Pembukaan nilai kredensial memerlukan alasan operasional
   yang dicatat ke dalam audit log keamanan sistem (`service.credential.reveal`).
5. **Dokumentasi Foto**: Unggah foto intake, before, after, testing (validasi MIME JPEG/PNG/WebP,
   maks 5 MB) dengan opsi publik untuk tracking pelanggan.
6. **Cetak Label Stiker Thermal**: Format stiker 58 mm dan 80 mm lengkap dengan nomor
   tiket besar, nama toko, info HP, keluhan, dan QR code yang mengarah ke link
   pelacakan publik.
7. **Daftar & Detail Servis**: Filter status, teknisi, rentang tanggal, pagination,
   serta timeline riwayat perpindahan status dari penerimaan hingga selesai.

---

## 2. File yang Ditambahkan / Diubah

### Backend

```
backend/internal/modules/service/credential.go      # Enkripsi AES-256 & audited reveal
backend/internal/modules/service/customer.go        # CRUD pelanggan, normalisasi nomor, devices
backend/internal/modules/service/handler.go         # HTTP handler endpoints
backend/internal/modules/service/model.go           # DTO & domain models
backend/internal/modules/service/number.go          # Sequence generator nomor tiket
backend/internal/modules/service/orders.go          # Service orders CRUD, filter, timeline
backend/internal/modules/service/photo.go           # Upload multipart & delete foto
backend/internal/modules/service/routes.go          # Wiring routes customer & service-orders
backend/internal/modules/service/service.go         # Core service & tenant boundaries
backend/internal/pkg/version/version.go             # 0.5.0
backend/tests/integration/service_test.go           # Acceptance integration tests
```

### Frontend

```
frontend/src/lib/api-client.ts                      # decodeEnvelope & api.getEnvelope
frontend/src/components/ui/dialog.tsx               # Modal dialog aksesibel
frontend/src/components/ui/select.tsx               # Styled select component
frontend/src/components/ui/textarea.tsx             # Styled textarea component
frontend/src/components/layout/nav-items.ts         # Unlock menu Servis & Pelanggan
frontend/src/app/router.tsx                         # Routing /services & /customers

frontend/src/features/customers/types.ts            # Tipe data pelanggan & perangkat
frontend/src/features/customers/api.ts              # API client pelanggan & perangkat
frontend/src/features/customers/components/
  ├── customer-form-dialog.tsx                      # Modal tambah/edit pelanggan
  └── customer-device-dialog.tsx                    # Modal tambah perangkat
frontend/src/features/customers/pages/
  ├── customer-list-page.tsx                        # Daftar & pencarian pelanggan
  └── customer-detail-page.tsx                      # Profil, perangkat & riwayat servis

frontend/src/features/services/types.ts             # Tipe data intake, order, foto, timeline
frontend/src/features/services/api.ts               # API client service orders & upload
frontend/src/features/services/components/
  ├── status-badge.tsx                              # 10 varian status badge + ikon
  ├── priority-badge.tsx                            # Badge normal & express
  ├── credential-reveal-dialog.tsx                  # Dialog buka PIN dengan alasan audit
  ├── credential-form-dialog.tsx                    # Dialog set/update/hapus kredensial
  ├── photo-upload-dialog.tsx                       # Dialog upload foto multi-stage
  └── service-edit-dialog.tsx                       # Dialog edit data penerimaan servis
frontend/src/features/services/pages/
  ├── service-list-page.tsx                         # Tabel servis, filter & pagination
  ├── service-create-page.tsx                       # Form penerimaan servis HP lengkap
  └── service-detail-page.tsx                       # Detail servis, foto, timeline, invoice

frontend/src/features/print/
  ├── thermal-label.tsx                             # Komponen stiker thermal 58/80 mm + QR
  └── print-label-modal.tsx                         # Modal pratinjau & print dialog
```

---

## 3. Fitur & Alur Operasional

### 3.1 Alur Intake Servis di Meja Kasir / Customer Service

1. Buka menu **Servis** &rarr; klik tombol **+ Servis Baru** (atau via route `/services/new`).
2. **Pilih Pelanggan**:
   - Jika pelanggan baru: masukkan Nama, No. WhatsApp (otomatis normalisasi format), dan Alamat.
   - Jika pelanggan lama: gunakan tombol "Cari Pelanggan Lama" untuk memilih data dari database.
3. **Data Perangkat**:
   - Isi Merek dan Model HP (contoh: Samsung Galaxy S22).
   - Masukkan IMEI/Nomor Seri jika tersedia.
   - Pilih kelengkapan yang disertakan (Unit HP, Charger, Casing, SIM Tray, dll.).
   - Catat kondisi fisik saat diterima (Layar retak, Lecet pemakaian, Bekas air, dll.).
4. **Kredensial Layar Kunci**:
   - Pilih jenis kunci: Tidak Ada, PIN Angka, Password, atau Pola.
   - Masukkan nilai kredensial. Nilai dienkripsi langsung menggunakan kunci `APP_ENC_KEY`
     dan tidak pernah dikirim balik dalam respons pembuatan/list order.
5. **Keluhan & Estimasi**:
   - Tulis keluhan utama pelanggan.
   - Masukkan estimasi biaya awal dan target tanggal penyelesaian.
   - Tentukan prioritas: Normal atau Express.
   - Tugaskan langsung ke teknisi atau biarkan kosong untuk masuk ke antrean umum.
6. **Foto Intake**:
   - Unggah hingga 5 foto kondisi fisik perangkat sebelum diservis.
7. **Penerbitan Tiket**:
   - Klik **Terbitkan Tiket Servis**. Sistem dalam satu transaksi database membuat
     nomor tiket atomik, status awal `received`, draft invoice bernomor, status timeline,
     dan token link tracking publik.
8. **Cetak Label Stiker**:
   - Klik **Cetak Label Tiket**. Modal menampilkan pratinjau label 58 mm atau 80 mm
     dengan barcode/QR code untuk ditempelkan pada fisik HP atau diberikan ke pelanggan.

### 3.2 Audit Log Kredensial Perangkat

- Demi melindungi data privasi pelanggan, nilai PIN/password perangkat tidak tampil di tabel maupun detail servis.
- Teknisi yang membutuhkan akses PIN (misal untuk testing layar setelah reparasi) menekan tombol **Buka Kredensial**.
- Modal mewajibkan input alasan operasional.
- Backend memvalidasi permission `service.credential.reveal`, mendeskripsikan nilai dengan AES-256-GCM,
  dan otomatis merekam aksi pembukaan ke tabel `audit_logs` bersama ID aktor, IP, dan alasan.

---

## 4. Hasil Pengujian & QA

### 4.1 Backend Integration Tests

```bash
cd backend
go test ./... -v
```

Hasil test `tests/integration/service_test.go`:
- `TestServiceCustomersLifecycleAndDuplicates`: Lulus (validasi format nomor HP, tolak duplikat per tenant, soft delete).
- `TestServiceOrderIntakeCredentialAndAudit`: Lulus (intake lengkap, enkripsi AES-256, penolakan reveal tanpa alasan, pencatatan audit log).
- `TestServiceTicketNumbersUniqueUnderConcurrency`: Lulus (20 request bersamaan menghasilkan nomor tiket sekuensial yang 100% unik tanpa tabrakan).
- `TestServiceOrderListFiltersAndTenantIsolation`: Lulus (filter status, teknisi, rentang tanggal, serta isolasi data antar tenant).
- `TestServicePhotoUploadValidationAndDelete`: Lulus (validasi ekstensi/MIME, batas ukuran file, dan hapus foto).

### 4.2 Frontend Static Check & Production Build

```bash
cd frontend
npm run typecheck
npm run build
```

- `tsc --noEmit`: Lulus tanpa warning/error TypeScript.
- `vite build`: Berhasil mem-build bundle produksi ke direktori `dist/`.

### 4.3 Pemeriksaan Integritas Dokumentasi

```bash
node scripts/check-docs.mjs
```

- 13 file Markdown: 0 broken link, seluruh code block seimbang.

---

## 5. Langkah Berikutnya (Tahap 6)

Dengan terselesaikannya modul penerimaan servis dan pelanggan pada Tahap 5,
pengembangan berlanjut ke **Tahap 6: Diagnosis, Pengerjaan & Status Machine**:
- Form diagnosis teknisi, catatan temuan, dan rekomendasi tindakan.
- Penambahan item biaya: Jasa servis & Sparepart dengan mutasi stok otomatis.
- Validasi 10 status state machine (`received` &rarr; `waiting_diagnosis` &rarr; `in_progress` &rarr; `completed` &rarr; `ready_for_pickup` &rarr; `picked_up`).
