# 15 — Implementasi Tahap 8: Invoice, Kasir & Pembayaran

**Status:** selesai ✅ **Versi aplikasi:** 0.8.0 ✅ **Tanggal:** 8 November 2026
**Acuan:** [`05 — Roadmap MVP`](05-roadmap-mvp.md) Tahap 8, [`06 — Kontrak API`](06-kontrak-api.md), [`07 — Keamanan`](07-keamanan.md)

---

## 1. Tujuan Tahap 8

Tahap 8 memastikan uang tercatat benar, dapat diaudit, dan bisa dicetak. Invoice
dibuat otomatis saat servis diterima, nilainya dikunci saat finalisasi, dan
setiap pembayaran mengubah status invoice secara konsisten. Kasir memperoleh
ringkasan harian, rekap metode bayar, serta aturan pickup yang mencegah HP
keluar sebelum tagihan selesai.

Cakupan yang diselesaikan:

1. **Invoice draft otomatis** untuk setiap service order baru.
2. **Diskon & catatan** nominal/persen dengan perhitungan ulang di server.
3. **Finalisasi invoice** dengan snapshot item immutable dan status otomatis.
4. **Pembayaran** tunai/transfer/QRIS/kartu/lainnya, DP parsial, validasi
   overpay, dan retry aman dengan `Idempotency-Key`.
5. **Void invoice & pembayaran** dengan alasan wajib dan audit.
6. **Kasir harian** dan rekap metode bayar per outlet.
7. **Cetak invoice A4** dan **struk thermal 58/80 mm** dari data server.
8. **Pickup wajib lunas** dengan override Owner yang tercatat.
9. **Pengeluaran dasar** dan kategori untuk laporan laba Tahap 10.

---

## 2. File yang Ditambahkan / Diubah

### Backend

```
backend/internal/modules/invoice/model.go            # DTO invoice, item, payment, store
backend/internal/modules/invoice/service.go          # List/detail, diskon, issue, void, print
backend/internal/modules/invoice/handler.go          # Handler HTTP + query parsing
backend/internal/modules/invoice/routes.go           # Route invoice
backend/internal/modules/payment/model.go            # DTO pembayaran & ringkasan harian
backend/internal/modules/payment/service.go          # Create/void payment, idempotency, daily summary
backend/internal/modules/payment/handler.go          # Handler HTTP + Idempotency-Key
backend/internal/modules/payment/routes.go           # Route pembayaran & kasir
backend/internal/modules/expense/model.go            # DTO pengeluaran & kategori
backend/internal/modules/expense/service.go          # CRUD pengeluaran, kategori, nomor dokumen
backend/internal/modules/expense/handler.go          # Handler HTTP + filter
backend/internal/modules/expense/routes.go           # Route pengeluaran
backend/internal/pkg/docnum/docnum.go                # Nomor dokumen atomik per outlet/hari
backend/internal/modules/service/orders.go           # Invoice draft otomatis saat intake
backend/internal/modules/service/status.go           # Aturan pickup lunas + override + audit
backend/internal/modules/service/model.go            # OverrideReason pada perubahan status
backend/internal/modules/service/handler.go          # Binding override_reason
backend/internal/router/router.go                    # Registrasi invoice/payment/expense
backend/internal/database/migrate.go                 # ExpectedVersion 16
backend/migrations/000016_payment_idempotency.*.sql  # Kolom + unique index idempotency
backend/tests/integration/invoice_test.go            # Integration test Tahap 8
backend/internal/pkg/version/version.go              # 0.8.0
```

### Frontend

```
frontend/src/features/invoices/types.ts                       # Tipe invoice, payment, filter
frontend/src/features/invoices/api.ts                         # Client API + link WhatsApp
frontend/src/features/invoices/format.ts                      # Format uang, status, metode bayar
frontend/src/features/invoices/components/
  ├── payment-dialog.tsx                                      # Pembayaran DP/lunas + metode
  ├── invoice-discount-dialog.tsx                             # Diskon nominal/persen
  ├── void-reason-dialog.tsx                                  # Void invoice/pembayaran
  └── invoice-status-badge.tsx                                # Badge status invoice
frontend/src/features/invoices/pages/
  ├── cashier-page.tsx                                        # Ringkasan harian + daftar pembayaran
  ├── invoice-list-page.tsx                                   # Daftar invoice, filter, pencarian
  └── invoice-detail-page.tsx                                 # Detail, aksi, pembayaran, cetak
frontend/src/features/expenses/types.ts                       # Tipe pengeluaran & kategori
frontend/src/features/expenses/api.ts                         # Client API pengeluaran
frontend/src/features/expenses/format.ts                      # Format metode & tanggal
frontend/src/features/expenses/components/
  ├── expense-form-dialog.tsx                                 # Tambah/ubah pengeluaran
  └── expense-category-dialog.tsx                             # Kategori pengeluaran
frontend/src/features/expenses/pages/expense-list-page.tsx    # Daftar, filter, total
frontend/src/features/print/invoice-a4.tsx                    # Template invoice A4
frontend/src/features/print/receipt-thermal.tsx               # Struk thermal 58/80 mm
frontend/src/features/print/print-invoice-modal.tsx           # Preview & print dua template
frontend/src/features/services/components/status-change-dialog.tsx # Override pickup
frontend/src/features/services/pages/service-detail-page.tsx  # Tautan ke invoice
frontend/src/app/router.tsx                                   # /cashier, /invoices, /expenses
frontend/src/components/layout/nav-items.ts                   # Menu Tahap 8 aktif
frontend/package.json                                         # 0.8.0
```

---

## 3. Perilaku Backend

### 3.1 Invoice draft otomatis

- `POST /service-orders` membuat satu invoice `draft` dalam transaksi yang sama
  dengan service order, memakai `invoice_no` dari pengaturan outlet
  (`invoice_prefix`/`invoice_format`, default `INV-YYMMDD-SEQ`).
- Respons intake memuat `invoice_no` sehingga kasir dapat langsung membuka
  invoice.
- `GET /service-orders/:id/invoice` mengembalikan detail invoice tanpa perlu
  mengetahui ID invoice.
- `GET /invoices` mendukung `q` (nomor invoice/tiket/nama/telepon), `status`
  (dipisah koma), `outlet_id`, `from`, `to`, `page`, dan `per_page` (maks 100).
  `from`/`to` menerima `YYYY-MM-DD` (zona waktu outlet) atau RFC3339.

### 3.2 Update invoice

`PATCH /invoices/:id` hanya berlaku saat invoice masih `draft`, `paid_amount`
0, dan `issued_at` kosong; selain itu `409 INVOICE_LOCKED`.

- `discount_type`: `percent` atau `amount`.
- `percent` maksimal 100; `amount` tidak boleh melebihi subtotal.
- `notes` dan `due_at` opsional; `due_at` kosong menghapus jatuh tempo.
- Server selalu menghitung ulang:
  `subtotal_service`, `subtotal_part`, `discount_amount`, dan
  `total_amount = subtotal_service + subtotal_part − discount_amount`.
  Pajak tetap 0 pada MVP.
- Diskon persen dibulatkan ke rupiah terdekat.
- Setiap perubahan menulis audit `invoice_update` dengan nilai sebelum/sesudah.

### 3.3 Finalisasi invoice

`POST /invoices/:id/issue`:

- Hanya dari status `draft`; percobaan kedua → `409 INVOICE_NOT_DRAFT`.
- Seluruh baris `service_order_items` disalin menjadi `invoice_items`
  (snapshot immutable: deskripsi, qty, harga, HPP, diskon baris, garansi).
- Total dihitung ulang dari snapshot; status menjadi `unpaid`.
- Invoice dengan total 0 langsung menjadi `paid`.
- Item invoice setelah finalisasi tidak berubah walaupun service order diubah.
- Audit `invoice_issue` mencatat nomor invoice, total, diskon, dan jumlah item.

### 3.4 Pembayaran

`POST /invoices/:id/payments` menerima:

```json
{
  "method": "cash",
  "amount": 300000,
  "paid_at": "2026-11-03T14:00:00+07:00",
  "reference_no": null,
  "notes": "DP"
}
```

- `method`: `cash`, `bank_transfer`, `qris`, `card`, `other`.
- `amount > 0` dan tidak melebihi sisa tagihan. Kelebihan → `409 OVERPAYMENT`
  dengan `details.remaining_amount`; invoice lunas juga `409 OVERPAYMENT`.
- Invoice harus sudah `issued` (`409 INVOICE_NOT_ISSUED`) dan bukan `void`
  (`409 INVOICE_VOID`).
- `paid_at` opsional (RFC3339, default sekarang) dan dipakai untuk laporan
  harian.
- Baris invoice dikunci `FOR UPDATE`; `paid_amount` dan status dihitung ulang
  dalam transaksi yang sama dengan insert payment dan audit `payment_create`.
- Status otomatis: `partial` bila belum menutup total, `paid` bila sudah.
- Nomor pembayaran `PAY-YYMMDD-SEQ` dibuat atomik per outlet per hari.
- `Idempotency-Key` opsional (maks 100 karakter):
  - retry dengan key sama mengembalikan payment pertama dengan HTTP `201`;
  - unique index parsial `(tenant_id, idempotency_key)` menjaga retry paralel;
  - key milik outlet yang tidak dapat diakses aktor → `409 IDEMPOTENCY_KEY_CONFLICT`.

Void pembayaran:

```json
{ "reason": "Salah input nominal" }
```

- Permission `payment.void`; `reason` wajib.
- Payment menjadi `void`, `paid_amount` dikurangi, status invoice dihitung ulang
  (`unpaid`/`partial`/`paid`), lalu audit `payment_void`.
- Payment yang sudah void → `409 PAYMENT_ALREADY_VOID`.

Void invoice:

- `POST /invoices/:id/void` dengan `reason` wajib dan permission `invoice.void`.
- Ditolak bila ada pembayaran posted → `409 INVOICE_HAS_PAYMENT`; void dua kali
  → `409 INVOICE_ALREADY_VOID`.
- Pembatalan service order otomatis mem-void invoice yang belum menerima uang
  (`draft`/`unpaid`).

### 3.5 Kasir harian & cetak

- `GET /payments` mendukung `q`, `status`, `method`, `outlet_id`, `from`, `to`,
  `page`, `per_page`. Pencarian mencakup nomor pembayaran, invoice, tiket,
  pelanggan, telepon, dan referensi.
- `GET /payments/daily-summary?date=YYYY-MM-DD&outlet_id=...` mengembalikan
  total pembayaran posted, jumlah transaksi, jumlah invoice, total void,
  serta rekap `methods` (nominal + jumlah).
- `GET /invoices/:id/print` mengembalikan
  `{ "template": "invoice", "paper": "A4", "invoice": { ... } }`. Frontend
  merender invoice A4 dan struk thermal 58/80 mm dari payload yang sama.
  Tahap ini belum menyediakan PDF dari server.

### 3.6 Pickup wajib lunas

- Transisi ke `picked_up` mensyaratkan invoice lunas; invoice total 0 dianggap
  lunas.
- Aktor tanpa `service.override_unpaid_pickup` → `409 UNPAID_PICKUP`.
- Pemegang permission (default hanya Owner) wajib mengirim `override_reason`;
  kosong → `422 VALIDATION_ERROR` pada field `override_reason`.
- Override dicatat pada:
  - audit `service_pickup_override` (invoice, total, dibayar, sisa, alasan,
    approver);
  - catatan riwayat status `Override pickup belum lunas: ...`.
- Pembatalan servis yang sudah memiliki pembayaran ditolak
  `409 INVOICE_LOCKED`.

### 3.7 Pengeluaran & kategori

- `GET /expenses` mendukung `q`, `category_id`, `outlet_id`, `from`, `to`,
  `page`, `per_page`; respons memuat `items` dan `total_amount`.
- `POST /expenses` memvalidasi kategori milik tenant, deskripsi, nominal > 0,
  metode (`cash`/`bank_transfer`/`qris`/`other`), dan tanggal `YYYY-MM-DD`.
  Nomor dokumen `EXP-YYMMDD-SEQ`.
- `PATCH /expenses/:id` parsial; `DELETE /expenses/:id` adalah soft delete.
- Semua mutasi menulis audit `expense_create`, `expense_update`, atau
  `expense_delete`.
- `GET /expense-categories` menyertakan `expense_count`; `POST` menambah
  kategori kustom dengan nama unik per tenant (case-insensitive) → duplikat
  `409 CATEGORY_EXISTS`. Belum ada endpoint ubah/hapus kategori.

### 3.8 Nomor dokumen atomik

`backend/internal/pkg/docnum` memakai
`INSERT ... ON CONFLICT ... DO UPDATE ... RETURNING` pada
`document_sequences`, sehingga nomor invoice, pembayaran, dan pengeluaran unik
meskipun dibuat bersamaan. Format mendukung `{PREFIX}`, `{YYYY}`, `{YYMMDD}`,
`{MM}`, `{DD}`, dan `{SEQ:n}`.

---

## 4. Frontend

| Route | Halaman | Permission |
|---|---|---|
| `/cashier` | Ringkasan harian, rekap metode, daftar pembayaran, void | `payment.view` |
| `/invoices` | Daftar invoice, filter status/tanggal, pencarian | `invoice.view` |
| `/invoices/:id` | Detail, diskon, finalisasi, pembayaran, void, cetak, WhatsApp | `invoice.view` |
| `/expenses` | Daftar pengeluaran, filter kategori/tanggal, total, CRUD | `expense.view` |

Perilaku UI:

- Tombol finalisasi, diskon, pembayaran, void, dan hapus hanya tampil bila
  pengguna memiliki permission terkait.
- Dialog pembayaran memvalidasi nominal terhadap sisa tagihan dan mengirim
  `Idempotency-Key` unik per percobaan.
- Detail invoice menampilkan snapshot item, riwayat pembayaran, status, dan
  aksi cetak. Tombol WhatsApp membuka deep link `wa.me` berisi ringkasan
  invoice.
- Modal cetak menyediakan tab Invoice A4 dan Struk, pilihan lebar 58/80 mm
  (default mengikuti `receipt_paper_width` pengaturan toko), serta pratinjau
  sebelum mencetak.
- Halaman detail servis menautkan invoice dan menampilkan dialog alasan
  override saat pickup belum lunas bagi pemegang permission.
- Semua tombol terhubung ke API nyata; tidak ada tombol mati.

---

## 5. Permission & Keamanan

| Permission | Kemampuan |
|---|---|
| `invoice.view` | Daftar/detail invoice, cetak |
| `invoice.create` | Finalisasi invoice |
| `invoice.update` | Diskon, catatan, jatuh tempo |
| `invoice.void` | Void invoice |
| `payment.view` | Daftar pembayaran, kasir harian, rekap metode |
| `payment.create` | Menerima pembayaran |
| `payment.void` | Void pembayaran |
| `expense.view` | Daftar pengeluaran & kategori |
| `expense.create` | Tambah pengeluaran/kategori |
| `expense.update` | Ubah pengeluaran |
| `expense.delete` | Hapus pengeluaran |
| `service.override_unpaid_pickup` | Override pickup belum lunas |

- Seed RBAC: Owner memiliki seluruh permission. Admin memiliki operasional
  invoice/pembayaran/pengeluaran tetapi **tidak** memiliki
  `service.override_unpaid_pickup`. Kasir memiliki
  `invoice.view/create/update` dan `payment.view/create`; pengeluaran hanya
  untuk Owner/Admin sesuai seed. Teknisi hanya melihat invoice/pembayaran bila
  diberi permission terkait.
- Semua query memakai `tenant_id`; invoice, pembayaran, dan pengeluaran juga
  dibatasi `outlet_id` yang dapat diakses aktor. Outlet lain → `404`/`403`.
- `cost_price` pada item invoice hanya dikembalikan bila aktor memiliki
  `inventory.cost.view`; selain itu bernilai 0.
- Setiap mutasi keuangan menulis audit log dengan pelaku dan nilai
  sebelum/sesudah.

---

## 6. Pengujian

Integration test baru di `backend/tests/integration/invoice_test.go`:

- `TestInvoiceLifecycleAndPayments` — draft otomatis, diskon, finalisasi
  snapshot, DP `partial`, pelunasan `paid`, overpay, void payment, void
  invoice, dan isolasi tenant.
- `TestCashierDailySummaryAndExpenses` — daftar pembayaran, ringkasan harian,
  rekap metode, CRUD pengeluaran, kategori, dan validasi.
- `TestPickupRequiresSettledInvoiceOrOwnerOverride` — penolakan pickup belum
  lunas, validasi alasan override, audit, dan pickup setelah lunas.
- `TestZeroTotalInvoiceIssuesAsPaid` — invoice total 0 langsung `paid` dan
  dapat di-pickup.

Hasil:

```bash
cd backend
gofmt -l internal/modules/invoice internal/modules/payment \
  internal/modules/expense internal/pkg/docnum internal/modules/service \
  internal/router/router.go tests/integration/invoice_test.go
go build ./...
go vet ./...
go test ./... -count=1
# PASS — seluruh paket, termasuk integration suite (±115 detik)

go test ./tests/integration -run "TestInvoice|TestCashier|TestPickup|TestZeroTotal" -count=1 -v
# PASS — 4 test Tahap 8

cd frontend
npm run typecheck
npm run build
# PASS — hanya peringatan ukuran chunk > 500 kB
```

`docs/openapi.yaml` diperbarui menjadi versi **0.8.0** dengan 13 path dan
schema baru untuk invoice, pembayaran, dan pengeluaran; total 55 path dan 108
schema, YAML berhasil diparse.

---

## 7. Batasan & Langkah Berikutnya

Batasan yang disengaja pada tahap ini:

- Cetak memakai browser (`window.print`) dari payload JSON; belum ada PDF
  server-side atau template HTML backend.
- Pajak masih 0 dan belum ada konfigurasi PPN.
- Diskon hanya di level invoice; diskon per baris item belum diekspos di UI.
- Bukti pembayaran/`proof_path` dan unggah struk belum diimplementasikan.
- Pengeluaran belum mendukung unggah nota atau persetujuan berjenjang.
- Kategori pengeluaran belum dapat diubah/dihapus.
- Belum ada laporan laba penuh; agregasi dikerjakan Tahap 10.
- Uji beban konkurensi pembayaran dan E2E Playwright dijadwalkan Tahap 11.

Tahap berikutnya adalah **Tahap 9 — Tracking Publik, Garansi & Notifikasi**:
halaman `/track`, token publik, QR label, garansi otomatis, klaim/rework,
notifikasi in-app + outbox WhatsApp, worker reminder, dan purge kredensial.
