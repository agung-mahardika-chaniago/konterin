# 14 — Implementasi Tahap 7: Inventori Sparepart Lengkap

**Status:** selesai ✅ **Versi aplikasi:** 0.7.0 ✅ **Tanggal:** 6 November 2026
**Acuan:** [`05 — Roadmap MVP`](05-roadmap-mvp.md) Tahap 7, [`06 — Kontrak API`](06-kontrak-api.md), [`07 — Keamanan`](07-keamanan.md)

---

## 1. Tujuan Tahap 7

Tahap 7 membuat stok sparepart akurat, mudah dicari, dan dapat diaudit. Semua
perubahan stok hanya terjadi melalui operasi yang menulis
`inventory_movements`, sehingga stok saat ini selalu dapat direkonsiliasi
dengan riwayatnya.

Cakupan yang diselesaikan:

1. **Master sparepart** — CRUD, SKU/barcode unik per tenant, kategori,
   supplier, model kompatibel, lokasi rak, dan status aktif.
2. **Pencarian & filter** — nama, SKU, barcode, model kompatibel, kategori,
   supplier, stok menipis, status aktif, dan pagination.
3. **Pembelian stok** — banyak baris dalam satu referensi, supplier opsional,
   nomor referensi, tanggal pembelian, dan total biaya.
4. **Penyesuaian stok (opname)** — stok fisik baru dengan alasan wajib dan
   catatan opsional.
5. **Riwayat movement** — per sparepart maupun global, filter tipe/outlet/
   tanggal, append-only, dengan pelaku dan stok sebelum/sesudah.
6. **Alert, valuasi, dan pemakaian** — stok ≤ minimum, nilai persediaan
   `Σ(stok × harga modal)`, dan sparepart paling banyak dipakai.
7. **Barcode lookup** — endpoint khusus dan pencarian form item servis yang
   mencocokkan barcode/SKU untuk scanner USB/HT.

---

## 2. File yang Ditambahkan / Diubah

### Backend

```
backend/internal/modules/inventory/model.go         # DTO, filter, request, actor
backend/internal/modules/inventory/service.go       # Master sparepart + validasi
backend/internal/modules/inventory/masterdata.go    # Kategori & supplier
backend/internal/modules/inventory/stock.go         # Pembelian, opname, movement, alert, valuasi
backend/internal/modules/inventory/handler.go       # Handler HTTP + parsing query
backend/internal/modules/inventory/routes.go        # Route inventori
backend/internal/router/router.go                   # Registrasi modul inventory
backend/internal/modules/service/items.go           # Pencarian sparepart termasuk barcode
backend/tests/integration/inventory_test.go         # Integration test lifecycle & isolasi
backend/internal/pkg/version/version.go             # 0.7.0
```

Tidak ada migrasi baru: tabel `spareparts`, `sparepart_categories`,
`suppliers`, dan `inventory_movements` sudah tersedia sejak migrasi
`000007_spareparts_inventory`. Index unik parsial `(tenant_id, sku)` dan
`(tenant_id, barcode)` untuk baris yang belum dihapus juga sudah ada.

### Frontend

```
frontend/src/features/inventory/types.ts                     # Tipe sparepart, movement, alert
frontend/src/features/inventory/api.ts                       # Client API inventori
frontend/src/features/inventory/format.ts                    # Label tipe movement & tanggal
frontend/src/features/inventory/components/
  ├── sparepart-form-dialog.tsx                              # Form tambah/ubah sparepart
  ├── master-data-dialog.tsx                                 # Kategori & supplier
  └── adjustment-dialog.tsx                                  # Opname stok
frontend/src/features/inventory/pages/
  ├── sparepart-list-page.tsx                                # Daftar, filter, statistik
  ├── sparepart-detail-page.tsx                              # Detail + riwayat stok
  ├── purchase-page.tsx                                      # Pembelian multi-baris
  └── movement-list-page.tsx                                 # Riwayat movement global
frontend/src/app/router.tsx                                  # Route /spareparts, /inventory/*
frontend/src/components/layout/nav-items.ts                  # Menu Tahap 7 aktif
frontend/src/features/services/components/items-editor.tsx   # Placeholder barcode
frontend/package.json                                        # 0.7.0
```

---

## 3. Perilaku Backend

### 3.1 Master sparepart

- `GET /spareparts` mendukung `q`, `category_id`, `supplier_id`,
  `low_stock`, `is_active`, `page`, dan `per_page` (maksimum 100).
- `q` mencari nama, SKU, barcode, dan model kompatibel (ILIKE).
- `POST /spareparts` memvalidasi SKU/nama wajib, kategori/supplier milik
  tenant, serta duplikat SKU/barcode case-insensitive. Stok awal > 0 menulis
  movement `initial` dengan `reference_type = opname` pada outlet aktif.
- `PATCH /spareparts/:id` hanya mengubah master data; `stock` tidak dapat
  diubah lewat endpoint ini. String kosong pada `category_id`, `supplier_id`,
  atau `barcode` menghapus nilai tersebut.
- `DELETE /spareparts/:id` adalah soft delete. Sparepart hilang dari daftar
  dan pencarian, tetapi movement serta referensi servis lama tetap utuh.
- `GET /spareparts/barcode/:code` mencocokkan barcode terlebih dahulu, lalu
  SKU sebagai fallback scanner.

### 3.2 Pembelian stok

`POST /inventory/purchases` menerima:

```json
{
  "supplier_id": "...",
  "reference_no": "INV-SUP-001",
  "purchased_at": "2026-11-03T10:00:00+07:00",
  "items": [
    { "sparepart_id": "...", "qty": 10, "unit_cost": 450000 }
  ],
  "notes": ""
}
```

- Minimal satu item; sparepart duplikat dalam satu request ditolak.
- Seluruh baris memakai satu `reference_id` dengan
  `reference_type = purchase` dan movement `purchase_in`.
- Stok bertambah sebesar `qty`. `cost_price` sparepart diperbarui ke
  `unit_cost` terbaru (kebijakan MVP: harga terakhir; weighted average
  direncanakan pasca-MVP).
- `purchased_at` opsional (RFC3339, default sekarang) dan menjadi waktu
  movement.
- Respons memuat `reference_id`, `purchased_at`, `total_cost`, dan movement
  per item.
- Sparepart nonaktif atau milik tenant lain ditolak.

### 3.3 Penyesuaian stok (opname)

`POST /inventory/adjustments` menerima `sparepart_id`, `new_stock`, `reason`
(wajib), dan `notes` opsional.

- `new_stock` adalah stok fisik hasil hitung, bukan delta.
- Movement `adjustment_in`/`adjustment_out` ditulis dengan
  `reference_type = opname`, alasan pada `notes`, dan `unit_cost` dari harga
  modal saat ini.
- Stok negatif ditolak validasi dan constraint database.
- Jika `new_stock` sama dengan stok saat ini → `409 STOCK_UNCHANGED`.
- Baris sparepart dikunci (`FOR UPDATE`) selama transaksi agar delta benar.

### 3.4 Riwayat movement

- `GET /inventory/movements` mendukung `sparepart_id`, `movement_type`,
  `outlet_id`, `from`, `to`, `page`, dan `per_page`.
- `GET /spareparts/:id/movements` memakai filter yang sama untuk satu part.
- `from`/`to` menerima `YYYY-MM-DD` (zona Asia/Jakarta) atau RFC3339.
- Movement bersifat append-only dan selalu memiliki `outlet_id`, pelaku,
  serta stok sebelum/sesudah.
- Aktor non-owner/admin hanya melihat movement dari outlet yang dapat
  diakses; permintaan outlet di luar akses menghasilkan
  `403 OUTLET_FORBIDDEN`.
- Tipe yang dikenal: `initial`, `purchase_in`, `service_out`,
  `service_return`, `adjustment_in`, `adjustment_out`, `transfer_in`,
  `transfer_out`.

### 3.5 Alert, valuasi, dan pemakaian

- `GET /inventory/alerts` mengembalikan sparepart aktif dengan
  `stock <= min_stock`, diurutkan berdasarkan kekurangan terbesar.
- `GET /inventory/valuation` menghitung `total_value = Σ(stock × cost_price)`
  untuk sparepart aktif, beserta `total_items` dan `total_stock`.
- `GET /inventory/top-used` meranking pemakaian `service_out`; default 30
  hari terakhir, `limit` 1–50 (default 10), dibatasi outlet aktor.

### 3.6 Kategori & supplier

- Kategori unik per tenant berdasarkan nama (case-insensitive) dan menyertakan
  jumlah sparepart aktif.
- Supplier mendukung pencarian nama/kontak/telepon.
- `POST` kategori/supplier memerlukan `sparepart.create`; `PATCH` memerlukan
  `sparepart.update`.
- Belum ada endpoint hapus kategori/supplier pada tahap ini.

---

## 4. Frontend

| Route | Halaman | Permission |
|---|---|---|
| `/spareparts` | Daftar sparepart, filter, alert, valuasi, top-used | `sparepart.view` |
| `/spareparts/:id` | Detail, informasi master, riwayat stok, aksi | `sparepart.view` |
| `/inventory/purchases/new` | Form pembelian multi-baris | `inventory.purchase` |
| `/inventory/movements` | Riwayat movement global + filter | `inventory.view` |

Perilaku UI:

- Tombol tambah/ubah/hapus, pembelian, dan penyesuaian hanya tampil bila
  pengguna memiliki permission terkait.
- Kolom harga modal, nilai stok, dan `unit_cost` movement hanya tampil untuk
  `inventory.cost.view`.
- Form sparepart memakai stok awal hanya saat membuat part baru; saat mengubah,
  stok diarahkan ke pembelian/opname.
- Dialog kategori/supplier dapat menambah dan mengubah data tanpa berpindah
  halaman.
- Pencarian form item servis menyebut barcode sehingga scanner USB/HT dapat
  langsung memilih sparepart.
- Semua tombol terhubung ke API nyata; tidak ada tombol mati.

---

## 5. Permission & Privasi Biaya

| Permission | Kemampuan |
|---|---|
| `sparepart.view` | Daftar/detail, kategori, supplier, alert, pencarian |
| `sparepart.create` | Tambah sparepart, kategori, supplier |
| `sparepart.update` | Ubah sparepart, kategori, supplier |
| `sparepart.delete` | Soft delete sparepart |
| `inventory.view` | Riwayat movement |
| `inventory.purchase` | Pembelian stok |
| `inventory.adjust` | Opname stok |
| `inventory.cost.view` | Harga modal, nilai stok, `unit_cost` movement, valuasi |
| `report.view` | Sparepart paling banyak dipakai |

- Seed RBAC: owner/admin memiliki seluruh permission inventori; teknisi dan
  kasir hanya `sparepart.view` + `inventory.view`.
- `cost_price`, `stock_value`, `unit_cost`, dan `total_value` disamarkan
  menjadi `0`/tidak dapat diakses bagi aktor tanpa `inventory.cost.view`.
- Semua query memakai `tenant_id`; sparepart bersifat tenant-level sedangkan
  movement terikat outlet.
- Setiap mutasi master, pembelian, dan opname menulis audit log.

---

## 6. Pengujian

Integration test baru di `backend/tests/integration/inventory_test.go`:

- `TestInventoryLifecycle` — kategori, supplier, sparepart + stok awal,
  pencarian, barcode, update, pembelian, perubahan harga modal, opname,
  alert, valuasi, top-used, movement global, dan soft delete.
- `TestInventoryPermissionsAndTenantIsolation` — penyamaran biaya untuk
  teknisi, penolakan pembelian, penolakan valuasi, dan isolasi tenant lain.

Hasil:

```bash
cd backend
go test ./tests/integration -run "TestInventory" -count=1 -v
# PASS — 2 test, ±12,8 detik
```

Pemeriksaan lain:

```bash
cd backend
gofmt -l internal/modules/inventory internal/router/router.go
go build ./...
go vet ./...
go test ./... -count=1

cd frontend
npm run typecheck
npm run build
```

`docs/openapi.yaml` memuat 42 path dan 80 schema; YAML berhasil diparse.

---

## 7. Batasan & Langkah Berikutnya

Batasan yang disengaja pada tahap ini:

- Import/export CSV sparepart tetap opsional dan belum diimplementasikan.
- Belum ada tabel dokumen pembelian terpisah; baris pembelian dikelompokkan
  melalui `reference_id` pada movement.
- Harga modal memakai kebijakan harga terakhir, belum weighted average.
- Transfer antar-outlet dan penghapusan kategori/supplier belum diekspos.
- Uji konkurensi stok yang lebih berat dan E2E Playwright dijadwalkan Tahap 11.

Tahap berikutnya adalah **Tahap 8 — Invoice, Kasir & Pembayaran**: invoice
draft/final, pembayaran tunai/transfer/QRIS/kartu, DP parsial, void dengan
alasan, status invoice otomatis, cetak invoice/struk, dan laporan transaksi.
