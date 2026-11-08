# 06 — Kontrak API

**Versi:** 1.0 (Tahap 1) · **Base URL:** `/api/v1`
**Format:** JSON UTF-8 · **Autentikasi:** `Authorization: Bearer <access_token>`

Dokumen ini adalah kontrak acuan. Spesifikasi OpenAPI 3.1 lengkap
(`docs/openapi.yaml`) dihasilkan pada Tahap 3 dan diperbarui setiap tahap.

---

## 1. Konvensi

### 1.1 Header

| Header | Wajib | Keterangan |
|---|---|---|
| `Authorization` | Ya (kecuali publik) | `Bearer <jwt>` |
| `X-Outlet-Id` | Untuk pengguna multi-outlet | Outlet aktif; divalidasi terhadap akses |
| `X-Request-Id` | Tidak | Diteruskan/dibuat server, dikembalikan di response |
| `Idempotency-Key` | Untuk pembayaran | UUID unik per percobaan; mencegah double charge |
| `Accept-Language` | Tidak | Default `id` |

### 1.2 Format data

| Tipe | Format | Contoh |
|---|---|---|
| Waktu | ISO-8601 UTC | `2026-11-03T04:15:00Z` |
| Tanggal | `YYYY-MM-DD` | `2026-11-03` |
| Uang | integer rupiah | `150000` |
| ID | UUID | `019a...` |
| Boolean | JSON boolean | `true` |
| Null | JSON `null` | |

### 1.3 Pagination & filter

Query umum: `page` (default 1), `per_page` (default 20, maks 100),
`q`, `sort` (whitelist per endpoint), `order` (`asc|desc`),
`outlet_id`, `from`, `to`.

```json
{
  "success": true,
  "data": [ ... ],
  "meta": { "page": 1, "per_page": 20, "total": 143, "total_pages": 8 }
}
```

### 1.4 Envelope error

```json
{
  "success": false,
  "error": {
    "code": "INSUFFICIENT_STOCK",
    "message": "Stok sparepart tidak mencukupi.",
    "details": { "sparepart_id": "...", "available": 2, "requested": 5 }
  },
  "request_id": "01JABCDEF..."
}
```

Validasi field:

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Data yang dikirim tidak valid.",
    "details": {
      "fields": {
        "customer.phone": "Nomor WhatsApp wajib diisi.",
        "items.0.qty": "Jumlah harus lebih dari 0."
      }
    }
  },
  "request_id": "..."
}
```

### 1.5 HTTP status & kode error

| HTTP | Kode | Kapan |
|---|---|---|
| 400 | `BAD_REQUEST` | Body/query tidak dapat diproses |
| 401 | `UNAUTHORIZED` | Token tidak ada/invalid |
| 401 | `TOKEN_EXPIRED` | Access token kedaluwarsa (klien refresh) |
| 401 | `INVALID_CREDENTIALS` | Login gagal (pesan generik) |
| 403 | `FORBIDDEN` | Tidak punya permission |
| 403 | `OUTLET_FORBIDDEN` | Outlet di luar akses |
| 404 | `NOT_FOUND` | Resource tidak ada / bukan milik tenant |
| 409 | `CONFLICT` | Duplikat data |
| 409 | `DUPLICATE_PHONE` | Nomor pelanggan sudah terdaftar |
| 409 | `INSUFFICIENT_STOCK` | Stok tidak cukup |
| 409 | `SPAREPART_INACTIVE` | Sparepart dinonaktifkan |
| 409 | `OVERPAYMENT` | Pembayaran melebihi sisa |
| 409 | `INVOICE_LOCKED` | Invoice tidak dapat diubah |
| 409 | `INVALID_STATUS_TRANSITION` | Transisi status ilegal |
| 409 | `ALREADY_PAID` | Servis sudah lunas |
| 422 | `VALIDATION_ERROR` | Validasi field |
| 422 | `UNPAID_INVOICE` | Pickup tanpa lunas (butuh override) |
| 429 | `RATE_LIMITED` | Terlalu banyak request |
| 500 | `INTERNAL_ERROR` | Kesalahan tak terduga (tanpa detail internal) |
| 503 | `SERVICE_UNAVAILABLE` | DB/maintenance |

Semua error 5xx dicatat dengan `request_id`; klien menampilkan pesan generik +
kode untuk dukungan.

---

## 2. Autentikasi

| Method | Path | Permission | Keterangan |
|---|---|---|---|
| POST | `/auth/login` | publik (rate limit) | Body: `email`, `password`; set cookie refresh |
| POST | `/auth/refresh` | cookie | Rotasi refresh token |
| POST | `/auth/logout` | login | Cabut sesi aktif |
| GET | `/auth/me` | login | Profil + role + permissions + outlets |
| PATCH | `/auth/profile` | login | Nama, telepon, avatar |
| POST | `/auth/change-password` | login | Password lama + baru; cabut sesi lain |

Contoh login:

```http
POST /api/v1/auth/login
Content-Type: application/json

{ "email": "owner@toko.id", "password": "••••••••" }
```

```json
{
  "success": true,
  "data": {
    "access_token": "eyJ...",
    "expires_in": 900,
    "user": {
      "id": "...",
      "name": "Budi",
      "email": "owner@toko.id",
      "role": "owner",
      "tenant_id": "...",
      "permissions": ["service.view", "service.create", "..."],
      "outlets": [ { "id": "...", "name": "Pusat", "code": "PST" } ],
      "active_outlet_id": "..."
    }
  }
}
```

---

## 3. Pengguna, Role, Outlet

| Method | Path | Permission |
|---|---|---|
| GET | `/users` | `user.view` |
| POST | `/users` | `user.manage` |
| GET | `/users/:id` | `user.view` |
| PATCH | `/users/:id` | `user.manage` |
| DELETE | `/users/:id` | `user.manage` |
| POST | `/users/:id/reset-password` | `user.manage` |
| PUT | `/users/:id/outlets` | `user.manage` |
| GET | `/roles` | `user.view` |
| GET | `/permissions` | `user.view` |
| GET | `/outlets` | `outlet.view` |
| POST | `/outlets` | `outlet.manage` |
| PATCH | `/outlets/:id` | `outlet.manage` |

Aturan:
- Admin tidak dapat membuat/mengubah pengguna role `owner` atau `super_admin`.
- Pengguna tidak dapat menonaktifkan dirinya sendiri.
- Reset password mengembalikan password sementara **sekali** dan mencabut sesi.

---

## 4. Dashboard

| Method | Path | Permission |
|---|---|---|
| GET | `/dashboard/summary?outlet_id&date` | `dashboard.view` |
| GET | `/dashboard/charts?from&to&outlet_id` | `dashboard.view` |
| GET | `/dashboard/alerts?outlet_id` | `dashboard.view` |
| GET | `/dashboard/technicians?outlet_id&from&to` | `dashboard.view` |

`summary` mengembalikan:

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

- `revenue_*` dihitung dari `payments` berstatus `posted`, bukan dari invoice.
- "Hari ini" mengikuti timezone outlet; `date/from/to` memakai zona outlet aktif.
- `total_customers` dan `low_stock` bersifat tingkat tenant; angka operasional
  lain mengikuti outlet aktif.

---

## 5. Pelanggan & Perangkat

| Method | Path | Permission |
|---|---|---|
| GET | `/customers?q&page&per_page` | `customer.view` |
| POST | `/customers` | `customer.create` |
| GET | `/customers/:id` | `customer.view` |
| PATCH | `/customers/:id` | `customer.update` |
| DELETE | `/customers/:id` | `customer.delete` |
| GET | `/customers/:id/services` | `customer.view` + `service.view` |
| GET | `/customers/:id/stats` | `customer.view` |
| GET | `/customers/:id/devices` | `customer.view` |
| POST | `/customers/:id/devices` | `customer.update` |

Create customer:

```json
{ "name": "Siti", "phone": "081234567890", "address": "Jl. ...", "notes": "" }
```

Normalisasi nomor: `08xx` → `628xx`; duplikat per tenant ditolak
`409 DUPLICATE_PHONE`.

`stats`: total servis, total transaksi (pembayaran posted), servis terakhir.

---

## 6. Servis

| Method | Path | Permission |
|---|---|---|
| GET | `/service-orders?q&status&technician_id&outlet_id&from&to&page` | `service.view` |
| POST | `/service-orders` | `service.create` |
| GET | `/service-orders/:id` | `service.view` |
| PATCH | `/service-orders/:id` | `service.update` |
| PATCH | `/service-orders/:id/status` | `service.status` |
| POST | `/service-orders/:id/assign` | `service.assign` |
| GET | `/service-orders/:id/timeline` | `service.view` |
| GET | `/service-orders/:id/public-link` | `service.view` |
| PUT | `/service-orders/:id/credential` | `service.update` |
| DELETE | `/service-orders/:id/credential` | `service.update` |
| POST | `/service-orders/:id/credential/reveal` | `service.credential.reveal` |
| POST | `/service-orders/:id/photos` | `service.update` |
| DELETE | `/service-orders/:id/photos/:photoId` | `service.update` |
| GET | `/service-orders/:id/diagnostic` | `diagnosis.view` |
| PUT | `/service-orders/:id/diagnostic` | `diagnosis.manage` |
| GET | `/service-orders/:id/items` | `service.view` |
| POST | `/service-orders/:id/items` | `service.update` |
| PATCH | `/service-orders/:id/items/:itemId` | `service.update` |
| DELETE | `/service-orders/:id/items/:itemId` | `service.update` |
| GET | `/spareparts/search?q&limit` | `service.view` |
| GET | `/service-orders/:id/invoice` | `invoice.view` |

### 6.1 Create service order

```json
{
  "customer": {
    "id": null,
    "name": "Siti",
    "phone": "081234567890",
    "address": "Jl. Merdeka 10"
  },
  "device": {
    "brand": "Samsung",
    "model": "Galaxy S22",
    "imei": "356789012345678",
    "color": "Phantom White",
    "completeness": "HP + casing, tanpa charger",
    "physical_condition": "Layar retak halus, body lecet"
  },
  "complaint": "Layar tidak mau menyala setelah jatuh",
  "credential": { "type": "pin", "value": "1234" },
  "estimated_cost": 750000,
  "estimated_done_at": "2026-11-04T17:00:00+07:00",
  "assigned_technician_id": "...",
  "priority": "normal",
  "notes": "Pelanggan minta dihubungi via WA",
  "internal_notes": ""
}
```

Sistem dalam satu transaksi:
1. Jika `customer.id` kosong, cari pelanggan berdasarkan nomor yang
   dinormalisasi; bila ditemukan, gunakan pelanggan tersebut (tidak membuat
   duplikat). Bila tidak ada, buat pelanggan baru.
2. Buat/temukan device berdasarkan IMEI (jika diisi) atau kombinasi
   pelanggan + merek + model.
3. Ambil nomor tiket dari sequence.
4. Enkripsi kredensial (jika ada).
5. Insert order + status history + audit.
6. Buat invoice draft bernomor (item kosong).

Catatan: endpoint `POST /customers` **menolak** nomor yang sudah terdaftar
(`409 DUPLICATE_PHONE`); reuse otomatis hanya terjadi pada alur intake servis.

Respons `201` berisi `id`, `ticket_no`, `public_token`, `tracking_url`,
`status`, `created_at`.
`credential` **tidak pernah** dikembalikan.

### 6.2 Ubah status

```http
PATCH /api/v1/service-orders/:id/status
{ "status": "in_progress", "notes": "Sparepart tiba" }
```

Status valid: `received`, `waiting_diagnosis`, `waiting_approval`,
`waiting_part`, `in_progress`, `testing`, `completed`, `ready_for_pickup`,
`picked_up`, `cancelled`. Transisi mengikuti peta BR-02; `picked_up` dan
`cancelled` bersifat terminal.

Aturan permission:
- Semua transisi memerlukan `service.status`.
- `cancelled` juga memerlukan `service.cancel` dan `cancel_reason` (atau
  `notes`) yang tidak kosong.
- Kasir hanya boleh melakukan `ready_for_pickup` dan `picked_up`.
- Teknisi tidak memiliki `service.assign`/`service.cancel`; mereka hanya dapat
  memindahkan status operasional.

Efek samping:
- `in_progress` → set `started_at` jika null; bila rollback dari `completed`,
  `completed_at` dikosongkan.
- `completed` → set `completed_at`.
- `picked_up` → set `picked_up_at`, `completed_at`, dan `credential_purge_at`
  (+30 hari).
- `cancelled` → mengembalikan stok sparepart beserta movement `service_return`
  dan mem-void invoice yang belum dibayar (`draft`/`unpaid`). Ditolak
  `409 INVOICE_LOCKED` bila invoice sudah `partial`/`paid` atau
  `paid_amount > 0`.

Setiap perubahan menulis `service_status_history` dan audit
`service_status_change`. Transisi ilegal → 409 `INVALID_STATUS_TRANSITION`.
Validasi pickup wajib lunas, aktivasi garansi, dan notifikasi WA menyusul
Tahap 8–9; saat ini transisi hanya mencatat timestamp.

### 6.3 Item servis

```json
// Jasa
{ "item_type": "service", "description": "Ganti LCD + pengerjaan",
  "qty": 1, "unit_price": 750000, "discount_amount": 0, "warranty_days": 30 }

// Sparepart
{ "item_type": "sparepart", "sparepart_id": "...", "qty": 1,
  "unit_price": 650000, "discount_amount": 0, "warranty_days": 90 }

// Biaya manual
{ "item_type": "custom", "description": "Ongkos kirim part",
  "qty": 1, "unit_price": 25000, "discount_amount": 0 }
```

- `item_type` = `service|sparepart|custom`; `qty` > 0, maks 9999. Qty
  sparepart wajib bilangan bulat.
- `unit_price` default harga jual untuk sparepart; untuk jasa/custom memakai
  nilai yang dikirim.
- Server menyimpan snapshot `cost_price`; nilai ini hanya dikembalikan bila
  aktor memiliki `inventory.cost.view` (selain itu `0`).
- Untuk sparepart, `description` opsional (diambil dari nama).
- `discount_amount` tidak boleh melebihi `qty × unit_price`.
- Server menghitung `subtotal` per baris, lalu memperbarui
  `subtotal_service`, `subtotal_part`, `total_amount` invoice `draft` dan
  `estimated_cost` order. Klien tidak mengirim total.
- Menambah sparepart memotong stok dengan `SELECT ... FOR UPDATE` dan movement
  `service_out`; stok kurang → `409 INSUFFICIENT_STOCK` dan item tidak
  tersimpan. Sparepart nonaktif → `409 SPAREPART_INACTIVE`.
- `PATCH` mengubah sebagian field. Perubahan qty sparepart menyesuaikan stok
  sebesar delta saja (tidak memotong ulang).
- `DELETE` mengembalikan stok sparepart + movement `service_return`.
- Item terkunci (`409 INVOICE_LOCKED`) bila invoice bukan `draft` atau sudah
  memiliki pembayaran.
- `GET /service-orders/:id` menyertakan `diagnostic` hanya untuk pemegang
  `diagnosis.view` dan item dengan `cost_price` tersamarkan sesuai permission.

### 6.4 Kredensial perangkat

Menyimpan/mengganti kredensial (write-only):

```http
PUT /api/v1/service-orders/:id/credential
{ "type": "pin", "value": "1234" }
```

- Nilai dienkripsi AES-256-GCM; server tidak pernah mengembalikannya.
- `type` = `pin|password|pattern`; `DELETE` pada path yang sama menghapus
  kredensial dan menulis audit.

Reveal (khusus teknisi yang ditugaskan / Owner):

```http
POST /api/v1/service-orders/:id/credential/reveal
{ "reason": "Perlu membuka perangkat untuk testing" }
```

- Wajib permission + order ditugaskan ke teknisi (atau Owner).
- Respons `{ "type": "pin", "value": "1234" }`, `Cache-Control: no-store`.
- Selalu menulis audit `reveal_credential` (tanpa nilai kredensial).
- Rate limit 5/menit/pengguna.

### 6.5 Foto

`multipart/form-data`: `file`, `stage` (`intake/before/after/testing/other`),
`is_public` (opsional). Maks 5 MB, `image/jpeg|png|webp`.

### 6.6 Diagnosis

```http
GET /api/v1/service-orders/:id/diagnostic
PUT /api/v1/service-orders/:id/diagnostic
```

```json
{
  "diagnosis": "Korsleting pada IC power",
  "findings": "Korosi di area charging IC",
  "action_taken": "Pembersihan ultrasonik dan penggantian IC",
  "test_result": "Belum dites ulang",
  "technician_notes": "Perlu pengeringan 2 jam",
  "estimated_done_at": "2026-11-08T10:00:00+07:00"
}
```

- `GET` memerlukan `diagnosis.view`; bila belum diisi respons `data` berupa
  objek kosong `{}`.
- `PUT` memerlukan `diagnosis.manage`, bersifat upsert, dan menulis audit
  `diagnostic_upsert`.
- `estimated_done_at` memakai RFC3339 dan ikut disinkronkan ke
  `service_orders.estimated_done_at`.
- Diagnosis tidak disertakan pada detail servis bila aktor tidak memiliki
  `diagnosis.view` (mis. kasir).

### 6.7 Pencarian sparepart untuk item

```http
GET /api/v1/spareparts/search?q=LCD&limit=20
```

- Memerlukan `service.view`.
- Mencari nama, SKU, atau model kompatibel pada sparepart aktif milik tenant.
- `limit` 1–50 (default 20).
- `cost_price` hanya terisi bila aktor memiliki `inventory.cost.view`.

---

## 7. Inventori

| Method | Path | Permission |
|---|---|---|
| GET | `/spareparts?q&category_id&supplier_id&low_stock&is_active&page&per_page` | `sparepart.view` |
| POST | `/spareparts` | `sparepart.create` |
| GET | `/spareparts/:id` | `sparepart.view` |
| PATCH | `/spareparts/:id` | `sparepart.update` |
| DELETE | `/spareparts/:id` | `sparepart.delete` |
| GET | `/spareparts/barcode/:code` | `sparepart.view` |
| GET | `/spareparts/:id/movements?from&to&page` | `inventory.view` |
| GET | `/inventory/movements?sparepart_id&movement_type&outlet_id&from&to&page` | `inventory.view` |
| POST | `/inventory/purchases` | `inventory.purchase` |
| POST | `/inventory/adjustments` | `inventory.adjust` |
| GET | `/inventory/alerts` | `sparepart.view` |
| GET | `/inventory/valuation` | `inventory.cost.view` |
| GET | `/inventory/top-used?from&to&limit` | `report.view` |
| GET | `/sparepart-categories` | `sparepart.view` |
| POST | `/sparepart-categories` | `sparepart.create` |
| PATCH | `/sparepart-categories/:id` | `sparepart.update` |
| GET | `/suppliers?q` | `sparepart.view` |
| POST | `/suppliers` | `sparepart.create` |
| PATCH | `/suppliers/:id` | `sparepart.update` |

### 7.1 Master sparepart

- `sku` dan `barcode` unik per tenant di antara sparepart yang belum dihapus.
- `GET /spareparts/barcode/:code` mencocokkan barcode lalu SKU (fallback
  scanner), hanya untuk sparepart yang belum dihapus.
- `PATCH` mengubah sebagian field. String kosong pada `category_id`,
  `supplier_id`, atau `barcode` menghapus relasi/nilai tersebut. `stock` tidak
  dapat diubah lewat `PATCH`; gunakan pembelian atau penyesuaian.
- `DELETE` adalah soft delete: sparepart hilang dari daftar/pencarian, tetapi
  riwayat movement dan referensi servis lama tetap utuh.
- `cost_price` dan `stock_value` bernilai `0` bila aktor tidak memiliki
  `inventory.cost.view`. `unit_cost` pada movement juga disamarkan.

### 7.2 Pembelian stok

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
- Semua baris memakai satu `reference_id` dengan `reference_type = purchase`
  dan movement `purchase_in`.
- Stok bertambah sebesar `qty`; `cost_price` sparepart diperbarui ke
  `unit_cost` terbaru (MVP harga terakhir, weighted average pasca-MVP).
- `purchased_at` opsional (RFC3339, default sekarang) dan menjadi waktu
  movement.
- Respons memuat `reference_id`, `total_cost`, dan movement per item.

### 7.3 Penyesuaian stok (opname)

```json
{ "sparepart_id": "...", "new_stock": 8,
  "reason": "Hasil opname mingguan", "notes": "" }
```

- `new_stock` adalah stok fisik hasil hitung, bukan delta.
- Movement `adjustment_in`/`adjustment_out` dengan `reference_type = opname`
  dan alasan wajib. Stok negatif ditolak validasi/`CHECK`.
- Jika `new_stock` sama dengan stok saat ini → `409 STOCK_UNCHANGED`.

### 7.4 Riwayat movement

- Movement bersifat append-only dan selalu memiliki `outlet_id`, pelaku, serta
  stok sebelum/sesudah.
- Filter `from`/`to` menerima `YYYY-MM-DD` (zona Asia/Jakarta) atau RFC3339.
- Aktor non-owner/admin hanya melihat movement dari outlet yang dapat diakses.
- Tipe: `initial`, `purchase_in`, `service_out`, `service_return`,
  `adjustment_in`, `adjustment_out`, `transfer_in`, `transfer_out`.

### 7.5 Alert, valuasi, dan pemakaian

- `GET /inventory/alerts` mengembalikan sparepart aktif dengan
  `stock <= min_stock`, diurutkan berdasarkan kekurangan terbesar.
- `GET /inventory/valuation` menghitung
  `total_value = Σ(stock × cost_price)` untuk sparepart aktif.
- `GET /inventory/top-used` meranking pemakaian `service_out`; default 30 hari
  terakhir dan `limit` 1–50 (default 10).

### 7.6 Kategori & supplier

- Kategori unik per tenant berdasarkan nama (case-insensitive).
- Supplier mendukung pencarian nama/kontak/telepon dan soft delete tidak
  diekspos pada tahap ini.
- `PATCH` kategori/supplier memerlukan `sparepart.update`.

---

## 8. Invoice & Pembayaran

| Method | Path | Permission |
|---|---|---|
| GET | `/invoices?q&status&outlet_id&from&to&page&per_page` | `invoice.view` |
| GET | `/invoices/:id` | `invoice.view` |
| PATCH | `/invoices/:id` | `invoice.update` |
| POST | `/invoices/:id/issue` | `invoice.create` |
| POST | `/invoices/:id/void` | `invoice.void` |
| GET | `/invoices/:id/print` | `invoice.view` |
| GET | `/service-orders/:id/invoice` | `invoice.view` |
| GET | `/invoices/:id/payments` | `payment.view` |
| POST | `/invoices/:id/payments` | `payment.create` |
| GET | `/payments?q&status&method&outlet_id&from&to&page&per_page` | `payment.view` |
| GET | `/payments/daily-summary?date&outlet_id` | `payment.view` |
| POST | `/payments/:id/void` | `payment.void` |

Status invoice: `draft`, `unpaid`, `partial`, `paid`, `void`.

### 8.1 Invoice draft otomatis

Setiap service order baru otomatis membuat satu invoice berstatus `draft`
dengan nomor mengikuti pengaturan invoice (default `INV-YYMMDD-SEQ`). Invoice
draft belum dapat dibayar; nilainya baru dikunci saat finalisasi.

`GET /service-orders/:id/invoice` memudahkan halaman detail servis membuka
invoice tanpa mengetahui ID invoice.

### 8.2 Update invoice (draft)

Hanya saat invoice masih `draft`, belum memiliki pembayaran, dan
`issued_at` masih kosong. Selain itu → `409 INVOICE_LOCKED`.

```json
{ "discount_type": "percent", "discount_value": 10,
  "notes": "Pelanggan langganan", "due_at": null }
```

- `discount_type`: `percent` atau `amount`.
- `percent` maksimal 100; `amount` tidak boleh melebihi subtotal → `422`.
- `due_at` menerima RFC3339; string kosong menghapus jatuh tempo.
- Total selalu dihitung ulang server:
  `total_amount = subtotal_service + subtotal_part − discount_amount`, pajak 0.
- Diskon persen dibulatkan ke rupiah terdekat.

### 8.3 Finalisasi invoice

`POST /invoices/:id/issue` (permission `invoice.create`):

- Hanya dapat dilakukan sekali dari status `draft` → selain itu
  `409 INVOICE_NOT_DRAFT`.
- Item `service_order_items` disalin menjadi snapshot immutable di
  `invoice_items`; perubahan item setelah finalisasi tidak mengubah invoice.
- Total dan diskon dihitung ulang; status menjadi `unpaid`.
- Invoice bernilai total 0 langsung menjadi `paid`.
- Snapshot item, total, dan status ditulis dalam satu transaksi + audit
  `invoice_issue`.

### 8.4 Pembayaran

```http
POST /api/v1/invoices/:id/payments
Idempotency-Key: 8f4c...
Content-Type: application/json

{
  "method": "cash",
  "amount": 300000,
  "paid_at": "2026-11-03T14:00:00+07:00",
  "reference_no": null,
  "notes": "DP"
}
```

Respons `201`:

```json
{
  "success": true,
  "data": {
    "payment": { "id": "...", "payment_no": "PAY-261103-0001", "amount": 300000,
                 "method": "cash", "status": "posted" },
    "invoice": { "total_amount": 750000, "paid_amount": 300000,
                 "remaining_amount": 450000, "status": "partial" }
  }
}
```

Aturan:
- `method`: `cash`, `bank_transfer`, `qris`, `card`, atau `other`.
- `amount > 0` dan ≤ sisa tagihan; jika melebihi → `409 OVERPAYMENT` dengan
  `details.remaining_amount`. Invoice yang sudah lunas juga `409 OVERPAYMENT`.
- `paid_at` opsional (RFC3339, default sekarang) dan menjadi dasar laporan
  harian.
- Invoice harus sudah difinalisasi (`INVOICE_NOT_ISSUED`) dan tidak `void`
  (`INVOICE_VOID`).
- `Idempotency-Key` opsional, maksimal 100 karakter. Permintaan ulang dengan
  key yang sama mengembalikan payment pertama dengan HTTP `201` tanpa membuat
  baris baru. Unique index parsial `(tenant_id, idempotency_key)` menjaga
  retry paralel. Key yang sudah dipakai pada outlet yang tidak dapat diakses
  aktor → `409 IDEMPOTENCY_KEY_CONFLICT`.
- Status invoice dihitung otomatis: `partial` bila `paid_amount < total_amount`,
  `paid` bila sudah menutup total.
- Void payment: `{ "reason": "..." }`, permission `payment.void`. Payment
  menjadi `void`, `paid_amount` dan status invoice dihitung ulang, lalu audit
  `payment_void`. Payment yang sudah `void` → `409 PAYMENT_ALREADY_VOID`.
- Void invoice hanya boleh bila belum ada pembayaran posted →
  `409 INVOICE_HAS_PAYMENT`; invoice void dua kali → `409 INVOICE_ALREADY_VOID`.

Format nomor pembayaran: `PAY-YYMMDD-SEQ` (atomik per outlet per hari).

### 8.5 Kasir harian, cetak, dan pickup

- `GET /payments/daily-summary?date=YYYY-MM-DD&outlet_id=...` mengembalikan
  `total_amount`, `posted_count`, `invoice_count`, `voided_amount`,
  `voided_count`, dan rekap `methods` (`cash`, `bank_transfer`, `qris`,
  `card`, `other`). Tanggal memakai zona waktu outlet; default hari ini.
- `GET /invoices/:id/print` mengembalikan payload
  `{ "template": "invoice", "paper": "A4", "invoice": { ... } }` yang dipakai
  UI untuk mencetak invoice A4 serta struk thermal 58/80 mm. Tahap ini belum
  menyediakan PDF dari server.
- Status servis `picked_up` mensyaratkan invoice lunas (atau total 0).
  Tanpa izin override → `409 UNPAID_PICKUP`. Owner/Admin dengan
  `service.override_unpaid_pickup` wajib mengirim `override_reason` (jika
  kosong → `422 VALIDATION_ERROR`); override dicatat sebagai audit
  `service_pickup_override` dan dicatat pada riwayat status.

---

## 9. Garansi & Klaim

| Method | Path | Permission |
|---|---|---|
| GET | `/warranties?status&q&expiring_in_days` | `warranty.view` |
| GET | `/warranties/:id` | `warranty.view` |
| POST | `/warranties/:id/claims` | `warranty.claim` |
| GET | `/claims?status` | `warranty.view` |
| GET | `/claims/:id` | `warranty.view` |
| PATCH | `/claims/:id` | `warranty.claim` |
| POST | `/claims/:id/rework` | `service.create` + `warranty.claim` |

Klaim:

```json
{ "complaint": "Layar kembali blank setelah 5 hari",
  "resolution": "", "status": "submitted" }
```

`POST /claims/:id/rework` membuat service order baru bertipe rework yang tertaut
ke klaim, tanpa biaya default, dan menyalin data pelanggan/perangkat.

---

## 10. Pengeluaran

| Method | Path | Permission |
|---|---|---|
| GET | `/expenses?q&from&to&category_id&outlet_id&page&per_page` | `expense.view` |
| POST | `/expenses` | `expense.create` |
| PATCH | `/expenses/:id` | `expense.update` |
| DELETE | `/expenses/:id` | `expense.delete` |
| GET | `/expense-categories` | `expense.view` |
| POST | `/expense-categories` | `expense.create` |

Semua data terikat `tenant_id` dan outlet aktif; filter `from`/`to` memakai
`expense_date` dengan format `YYYY-MM-DD`.

### 10.1 Mencatat pengeluaran

```json
{
  "category_id": "...",
  "description": "Beli tinta printer struk",
  "amount": 150000,
  "expense_date": "2026-11-07",
  "method": "cash",
  "notes": "Nota terlampir"
}
```

- `category_id` wajib milik tenant yang sama.
- `description` wajib, maksimal 200 karakter; `amount > 0`.
- `method`: `cash`, `bank_transfer`, `qris`, atau `other`.
- Nomor dokumen otomatis `EXP-YYMMDD-SEQ` (atomik per outlet per hari).
- Respons `201` memuat pengeluaran lengkap dengan nama kategori/outlet.
- Daftar mengembalikan `items` dan `total_amount` sesuai filter, diurutkan
  tanggal terbaru; soft delete tidak ditampilkan.

### 10.2 Ubah & hapus

- `PATCH /expenses/:id` bersifat parsial dan memvalidasi field yang dikirim.
- `DELETE /expenses/:id` adalah soft delete; data hilang dari daftar/laporan
  tetapi baris tetap tersimpan.
- Setiap create/update/delete menulis audit log (`expense_create`,
  `expense_update`, `expense_delete`).
- Pengeluaran outlet lain → `404`.

### 10.3 Kategori pengeluaran

- `GET /expense-categories` mengembalikan kategori sistem dan kustom beserta
  `expense_count` (hanya pengeluaran aktif).
- `POST /expense-categories` menambah kategori kustom; nama unik per tenant
  secara case-insensitive → `409 CATEGORY_EXISTS`.
- Belum ada endpoint ubah/hapus kategori pada tahap ini.

---

## 11. Laporan

| Method | Path | Permission |
|---|---|---|
| GET | `/reports/revenue?period=daily|monthly|yearly&from&to&outlet_id` | `report.view` |
| GET | `/reports/services?from&to&outlet_id` | `report.view` |
| GET | `/reports/brands?from&to&outlet_id` | `report.view` |
| GET | `/reports/damage-types?from&to&outlet_id` | `report.view` |
| GET | `/reports/service-vs-parts?from&to&outlet_id` | `report.view` |
| GET | `/reports/profit?from&to&outlet_id` | `report.profit.view` |
| GET | `/reports/expenses?from&to&outlet_id` | `report.profit.view` |
| GET | `/reports/technicians?from&to&outlet_id` | `report.view` |
| GET | `/reports/stock?outlet_id` | `report.view` |
| GET | `/reports/inventory-valuation?outlet_id` | `report.profit.view` |

Semua endpoint menerima `format=json|xlsx|pdf` (default `json`). Untuk
`xlsx/pdf`, respons berupa file unduhan dengan `Content-Disposition`.
Export **tidak** memuat kolom sensitif (PIN, catatan internal, harga modal
kecuali pengguna berhak).

Contoh respons revenue:

```json
{
  "success": true,
  "data": {
    "period": "daily",
    "from": "2026-11-01",
    "to": "2026-11-03",
    "totals": { "revenue": 7200000, "service": 4800000, "parts": 2400000,
                "transactions": 18 },
    "rows": [
      { "date": "2026-11-01", "revenue": 2400000, "service": 1600000,
        "parts": 800000, "transactions": 6 }
    ]
  }
}
```

---

## 12. Tracking Publik

Tidak memerlukan autentikasi. Rate limit 10/menit/IP.

| Method | Path | Keterangan |
|---|---|---|
| GET | `/public/track?ticket=FP-261103-0001&phone_last4=7890` | Pencarian aman |
| GET | `/public/track/:token` | Link langsung dari QR |
| GET | `/public/store` | Nama, logo, alamat, jam operasional (opsional) |

Respons (whitelist):

```json
{
  "success": true,
  "data": {
    "ticket_no": "FP-261103-0001",
    "store": { "name": "FixPhone", "logo_url": "..." },
    "status": "in_progress",
    "status_label": "Sedang Diperbaiki",
    "received_at": "2026-11-03T02:00:00Z",
    "estimated_done_at": "2026-11-04T10:00:00Z",
    "device": "Samsung Galaxy S22",
    "complaint": "Layar tidak menyala",
    "diagnosis": "LCD rusak, perlu penggantian",
    "invoice_status": "partial",
    "payment_status_label": "Belum Lunas",
    "pickup_status": "Belum Diambil",
    "warranty": { "active": false, "end_date": null }
  }
}
```

Tidak pernah memuat: nama lengkap pelanggan, nomor HP lengkap, alamat, IMEI,
PIN, catatan internal, foto, harga modal, nama teknisi.

---

## 13. Pengaturan

| Method | Path | Permission |
|---|---|---|
| GET | `/settings?outlet_id` | `settings.view` |
| PATCH | `/settings` | `settings.manage` |
| POST | `/settings/logo` | `settings.manage` |
| POST | `/settings/whatsapp/test` | `settings.manage` |
| GET | `/settings/backups` | `settings.manage` |
| POST | `/settings/backups` | `settings.manage` |
| GET | `/settings/backups/:id/download` | `settings.manage` |

Kredensial WhatsApp dikirim ke server, disimpan terenkripsi, dan **tidak pernah**
dikembalikan (hanya status `configured: true`).

---

## 14. Notifikasi & WhatsApp

| Method | Path | Permission |
|---|---|---|
| GET | `/notifications?unread=true` | login |
| PATCH | `/notifications/:id/read` | login |
| POST | `/notifications/read-all` | login |
| GET | `/whatsapp-messages?status&page` | `settings.manage` |
| POST | `/whatsapp-messages/:id/retry` | `settings.manage` |
| POST | `/service-orders/:id/notify` | `service.update` |

`notify` merender template dan memasukkan pesan ke outbox; respons memuat
`message_id` dan `status`.

---

## 15. Audit Log

| Method | Path | Permission |
|---|---|---|
| GET | `/audit-logs?user_id&module&action&entity_type&from&to&page` | `audit.view` |

Read-only; tidak ada endpoint create/update/delete.

---

## 16. Platform (Super Admin)

| Method | Path | Permission |
|---|---|---|
| GET | `/platform/tenants` | `platform.tenant.view` |
| POST | `/platform/tenants` | `platform.tenant.manage` |
| GET | `/platform/tenants/:id` | `platform.tenant.view` |
| PATCH | `/platform/tenants/:id` | `platform.tenant.manage` |
| POST | `/platform/tenants/:id/suspend` | `platform.tenant.manage` |
| POST | `/platform/tenants/:id/impersonate` | `platform.tenant.manage` (audit wajib) |

Super Admin tidak dapat membaca data bisnis tanpa impersonation yang tercatat
dan dibatasi waktu.

---

## 17. Health & Meta

| Method | Path | Auth |
|---|---|---|
| GET | `/healthz` | publik (tanpa detail DB) |
| GET | `/readyz` | publik/internal (DB ping, versi migrasi) |
| GET | `/version` | publik (versi build, commit) |

---

## 18. Contoh Lengkap: Alur Servis via API

```http
# 1. Login
POST /api/v1/auth/login
{ "email": "admin@toko.id", "password": "..." }

# 2. Buat pelanggan + servis
POST /api/v1/service-orders
Authorization: Bearer ...
{
  "customer": { "name": "Siti", "phone": "081234567890" },
  "device": { "brand": "Samsung", "model": "Galaxy S22" },
  "complaint": "Layar tidak menyala",
  "estimated_cost": 750000
}
→ 201 { "ticket_no": "FP-261103-0001", "status": "received" }

# 3. Diagnosis
PUT /api/v1/service-orders/{id}/diagnostic
{ "diagnosis": "LCD rusak", "findings": "Tidak ada gambar, backlight mati",
  "action_taken": "Ganti LCD", "estimated_done_at": "2026-11-04T10:00:00Z" }

# 4. Tambah jasa + sparepart
POST /api/v1/service-orders/{id}/items
{ "item_type": "service", "description": "Jasa ganti LCD",
  "qty": 1, "unit_price": 200000, "warranty_days": 30 }

POST /api/v1/service-orders/{id}/items
{ "item_type": "sparepart", "sparepart_id": "...", "qty": 1,
  "unit_price": 650000, "warranty_days": 90 }

# Ubah qty item (stok sparepart menyesuaikan delta)
PATCH /api/v1/service-orders/{id}/items/{item_id}
{ "qty": 2 }

# 5. Ubah status
PATCH /api/v1/service-orders/{id}/status
{ "status": "in_progress" }
PATCH /api/v1/service-orders/{id}/status
{ "status": "completed", "notes": "Testing 30 menit normal" }

# 6. Terbitkan invoice (finalisasi item)
POST /api/v1/invoices/{invoice_id}/issue

# 7. Pembayaran
POST /api/v1/invoices/{invoice_id}/payments
Idempotency-Key: ...
{ "method": "qris", "amount": 850000 }

# 8. Ambil perangkat
PATCH /api/v1/service-orders/{id}/status
{ "status": "picked_up" }

# 9. Cetak
GET /api/v1/invoices/{invoice_id}/print
GET /api/v1/service-orders/{id}/print-label
```

---

## 19. Catatan Implementasi

- Swagger/OpenAPI di-serve di `/api/docs` hanya pada `development`; production
  dinonaktifkan atau dilindungi permission.
- Semua endpoint `GET` daftar wajib memiliki index pendukung (lihat `03`).
- Perubahan kontrak API wajib memperbarui dokumen ini dan versi minor.
- Frontend mengandalkan `permissions` dari `/auth/me`; server tetap memeriksa
  setiap request.
- Endpoint yang mengembalikan file menggunakan `Content-Type` yang tepat dan
  `Content-Disposition: inline|attachment`.
