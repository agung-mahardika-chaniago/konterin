# 13 — Implementasi Tahap 6: Diagnosis, Pengerjaan, Status Machine & Item Jasa/Part

**Status:** selesai ✅ **Versi aplikasi:** 0.6.0 ✅ **Tanggal:** 6 November 2026
**Acuan:** [`05 — Roadmap MVP`](05-roadmap-mvp.md) Tahap 6, [`06 — Kontrak API`](06-kontrak-api.md), [`07 — Keamanan`](07-keamanan.md)

---

## 1. Tujuan Tahap 6

Tahap 6 melengkapi alur pengerjaan servis setelah perangkat diterima di counter:
teknisi mencatat diagnosis, menambahkan jasa/sparepart, memindahkan status sesuai
state machine, dan menugaskan teknisi. Seluruh perubahan dihitung dan divalidasi
server, bukan oleh klien.

Cakupan yang diselesaikan:

1. **Form diagnosis** — diagnosis, temuan, tindakan, hasil testing, catatan
   teknisi, dan estimasi selesai.
2. **Item jasa & sparepart** — tambah, ubah, hapus, diskon baris, garansi item,
   serta tipe `custom` untuk biaya manual.
3. **Mutasi stok otomatis** — pemakaian sparepart memotong stok, edit qty
   menyesuaikan delta, hapus item mengembalikan stok, semuanya dengan movement.
4. **State machine 10 status** — validasi transisi, permission per aksi,
   timestamp otomatis, riwayat status, dan audit.
5. **Penugasan teknisi & prioritas** — validasi teknisi aktif pada outlet yang
   sama, prioritas normal/express.
6. **Privasi data** — harga modal hanya untuk `inventory.cost.view`; diagnosis
   hanya untuk `diagnosis.view`; tenant/outlet tetap terisolasi.
7. **UI detail servis** — stepper status, dialog ubah status, form diagnosis,
   editor item dengan pencarian sparepart, dan dialog penugasan.

---

## 2. File yang Ditambahkan / Diubah

### Backend

```
backend/internal/modules/service/diagnostic.go      # Upsert & baca diagnosis
backend/internal/modules/service/items.go           # Item CRUD, stok, invoice totals
backend/internal/modules/service/status.go          # State machine & penugasan
backend/internal/modules/service/handler.go         # Handler HTTP Tahap 6
backend/internal/modules/service/model.go           # DTO, permission actor
backend/internal/modules/service/orders.go          # Embed diagnosis/item di detail
backend/internal/modules/service/routes.go          # Route diagnosis/item/assign
backend/internal/pkg/version/version.go             # 0.6.0
backend/tests/integration/repair_test.go            # 4 integration test baru
```

Tidak ada migrasi baru: tabel `service_diagnostics`, `service_order_items`,
`service_status_history`, `inventory_movements`, dan kolom timestamp sudah
tersedia sejak Tahap 2.

### Frontend

```
frontend/src/features/services/types.ts             # Tipe diagnosis, item, status
frontend/src/features/services/api.ts               # API diagnosis/item/assign/status
frontend/src/features/services/status-flow.ts       # Cermin state machine untuk UI
frontend/src/features/services/components/
  ├── status-stepper.tsx                            # Stepper progres servis
  ├── status-change-dialog.tsx                      # Dialog transisi + alasan batal
  ├── diagnosis-form.tsx                            # Form diagnosis teknisi
  ├── items-editor.tsx                              # Editor jasa/sparepart + pencarian
  └── assign-technician-dialog.tsx                  # Penugasan teknisi & prioritas
frontend/src/features/services/pages/service-detail-page.tsx
frontend/package.json                               # 0.6.0
```

---

## 3. Perilaku Backend

### 3.1 State machine & permission

Transisi mengikuti peta BR-02:

```
received          → waiting_diagnosis | cancelled
waiting_diagnosis → waiting_approval | waiting_part | in_progress | cancelled
waiting_approval  → waiting_part | in_progress | cancelled
waiting_part      → in_progress | cancelled
in_progress       → waiting_part | testing | cancelled
testing           → in_progress | completed | cancelled
completed         → ready_for_pickup | in_progress | cancelled
ready_for_pickup  → picked_up | completed | cancelled
picked_up         → terminal
cancelled         → terminal
```

Aturan yang ditegakkan:

- Semua transisi memerlukan `service.status`.
- Pembatalan memerlukan `service.cancel` dan alasan tidak kosong.
- Kasir hanya boleh `ready_for_pickup` dan `picked_up`.
- Pembatalan ditolak `409 INVOICE_LOCKED` bila invoice sudah `partial`/`paid`
  atau memiliki `paid_amount > 0`.
- `in_progress` mengisi `started_at`; rollback dari `completed` mengosongkan
  `completed_at`.
- `picked_up` mengisi `picked_up_at`, `completed_at`, dan
  `credential_purge_at` (+30 hari).
- Pembatalan mengembalikan stok sparepart dan mem-void invoice yang belum
  dibayar (`draft`/`unpaid`), sesuai BR-10.
- Setiap transisi menulis `service_status_history` dan audit
  `service_status_change`.

Penugasan teknisi memerlukan `service.assign`, memvalidasi bahwa target adalah
user role `technician` yang aktif dan bertugas di outlet order. Prioritas hanya
menerima `normal` atau `express`.

### 3.2 Diagnosis

- `GET /service-orders/:id/diagnostic` → `diagnosis.view`; mengembalikan `{}`
  bila belum ada.
- `PUT /service-orders/:id/diagnostic` → `diagnosis.manage`; upsert dan audit.
- `estimated_done_at` (RFC3339) disinkronkan ke `service_orders`.
- Detail servis hanya menyertakan diagnosis bila aktor memiliki
  `diagnosis.view`; kasir tidak menerima data ini.

### 3.3 Item & perhitungan

- `item_type` = `service`, `sparepart`, atau `custom`.
- `qty` > 0, maksimum 9999; qty sparepart wajib bilangan bulat.
- `subtotal = qty × unit_price − discount_amount`; diskon tidak boleh melebihi
  harga baris.
- Harga jual sparepart dipakai sebagai default `unit_price`; `cost_price`
  disimpan sebagai snapshot.
- Server memperbarui `subtotal_service`, `subtotal_part`, `total_amount` invoice
  `draft`, dan `estimated_cost` order setelah setiap perubahan item.
- Stok sparepart dipotong dengan `SELECT ... FOR UPDATE` di transaksi yang sama.
  Stok kurang → `409 INSUFFICIENT_STOCK`; sparepart nonaktif →
  `409 SPAREPART_INACTIVE`; item tidak tersimpan saat gagal.
- `PATCH` qty sparepart menyesuaikan stok sebesar delta saja, sehingga edit
  tidak memotong stok dua kali.
- `DELETE` mengembalikan stok dan mencatat movement `service_return`.
- Item terkunci (`409 INVOICE_LOCKED`) bila invoice bukan `draft` atau sudah
  memiliki pembayaran.
- `cost_price` dikembalikan hanya untuk pemegang `inventory.cost.view`;
  selain itu bernilai `0`.

### 3.4 Endpoint baru

| Method | Path | Permission |
|---|---|---|
| PATCH | `/service-orders/:id/status` | `service.status` (+ `service.cancel` untuk batal) |
| POST | `/service-orders/:id/assign` | `service.assign` |
| GET/PUT | `/service-orders/:id/diagnostic` | `diagnosis.view` / `diagnosis.manage` |
| GET/POST | `/service-orders/:id/items` | `service.view` / `service.update` |
| PATCH/DELETE | `/service-orders/:id/items/:itemId` | `service.update` |
| GET | `/spareparts/search?q&limit` | `service.view` |

---

## 4. Perilaku Frontend

Halaman detail servis kini menampilkan:

1. **Stepper status** — jalur utama `received → … → picked_up`, dengan badge
   untuk status cabang (`waiting_approval`, `waiting_part`) dan `cancelled`.
2. **Dialog ubah status** — hanya menampilkan transisi yang diizinkan untuk
   status saat ini, role, dan permission pengguna. Pembatalan mewajibkan
   alasan. Setelah sukses, detail dan timeline dimuat ulang.
3. **Form diagnosis** — field diagnosis, temuan, tindakan, hasil testing,
   catatan teknisi, dan estimasi selesai. Tombol simpan hanya muncul untuk
   `diagnosis.manage`.
4. **Editor item** — tambah/edit/hapus jasa, sparepart, atau custom; pencarian
   sparepart dengan stok dan harga; diskon baris; garansi. Editor menampilkan
   state terkunci saat invoice sudah final dan menyembunyikan aksi bila tidak
   berizin.
5. **Dialog penugasan** — pilih teknisi dari daftar nyata (`/technicians`) dan
   prioritas normal/express; hanya untuk `service.assign`.
6. **Penanganan state** — loading, empty, error, dan sukses ditampilkan pada
   setiap komponen; tombol aksi mengikuti permission dari `/auth/me`.

---

## 5. Hasil Pengujian & QA

### 5.1 Backend

```bash
cd backend
gofmt -w ./cmd ./internal ./migrations ./tests
go build ./...
go vet ./...
go test ./... -count=1
```

Hasil:

- `gofmt`, `go build ./...`, dan `go vet ./...`: lulus.
- Seluruh test: lulus, termasuk `tests/integration` (136,3 detik).

Test baru di `backend/tests/integration/repair_test.go`:

- `TestServiceRepairStatusMachineAndDiagnosis` — transisi ilegal ditolak,
  diagnosis tersimpan dan tersinkron ke order, seluruh jalur status sampai
  `picked_up`, timestamp benar, status terminal tidak dapat berubah, timeline
  lengkap.
- `TestServiceRepairPermissionsAndAssignment` — pembatalan wajib alasan,
  teknisi tidak boleh assign/cancel, owner assign + prioritas express, kasir
  hanya boleh transisi serah terima.
- `TestServiceRepairItemsStockAndInvoiceTotals` — subtotal server, stok turun
  tepat sekali, stok kurang ditolak atomik, edit qty menyesuaikan delta,
  hapus mengembalikan stok, invoice final mengunci item, harga modal hanya
  untuk owner.
- `TestServiceRepairCancellationVoidsUnpaidInvoiceAndRestocks` — pembatalan
  mem-void invoice `unpaid`, mengembalikan stok + movement `service_return`,
  dan invoice berbayar memblokir pembatalan dengan `INVOICE_LOCKED`.

### 5.2 Frontend

```bash
cd frontend
npm run typecheck
npm run build
```

- `tsc --noEmit`: lulus.
- `vite build`: berhasil. Peringatan ukuran chunk (>500 kB) masih ada dan
  menjadi catatan optimasi Tahap 11.

### 5.3 Validasi OpenAPI

```bash
node -e "const fs=require('fs'); const YAML=require('./frontend/node_modules/yaml'); YAML.parse(fs.readFileSync('docs/openapi.yaml','utf8'));"
```

- `docs/openapi.yaml` valid; 28 path terdaftar, termasuk status, diagnosis,
  item, dan pencarian sparepart.

---

## 6. Keamanan & Isolasi Tenant

- Semua query item, diagnosis, status, dan penugasan memakai `tenant_id` dan
  memvalidasi akses outlet aktor melalui `findOrder`.
- `GET` detail tidak membocorkan diagnosis tanpa `diagnosis.view`.
- `cost_price` tidak pernah dikirim ke aktor tanpa `inventory.cost.view`,
  termasuk pada hasil pencarian sparepart.
- Pembatalan tidak dapat menghapus jejak uang: invoice berbayar memblokir
  pembatalan.
- Setiap mutasi item/status/diagnosis/penugasan menulis audit log.
- Permission diperiksa di route (`can(...)`) dan aturan tambahan di service
  (kasir, pembatalan, validasi teknisi).

---

## 7. Batasan & Langkah Berikutnya

Batasan yang disengaja pada tahap ini:

- Validasi pickup wajib lunas, override owner, aktivasi garansi, dan notifikasi
  WhatsApp belum diimplementasikan; akan ditangani pada Tahap 8–9.
- Filter "teknisi hanya melihat order yang ditugaskan" sudah aktif melalui
  `business_settings.technician_sees_only_assigned` dan diterapkan pada daftar
  serta detail servis.
- Foto before/after/testing memakai dialog upload Tahap 5; belum ada galeri
  khusus per tahap pengerjaan.
- E2E Playwright dan uji konkurensi stok yang lebih berat dijadwalkan Tahap 11.

Tahap berikutnya adalah **Tahap 7 — Inventori Sparepart Lengkap**: CRUD
sparepart/kategori/supplier, pembelian stok, opname, riwayat movement, alert
stok minimum, dan valuasi inventori. Tahap 7 telah selesai; lihat
[`14 — Implementasi Tahap 7`](14-implementasi-tahap-7.md).
