# 07 — Keamanan & Kontrol Akses

**Versi:** 1.0 (Tahap 1) · **Acuan:** OWASP ASVS L1/L2 (disesuaikan), UU PDP No. 27/2022

---

## 1. Prinsip

1. **Least privilege** — pengguna hanya mendapat izin yang dibutuhkan.
2. **Defense in depth** — validasi klien, server, database constraint, RLS.
3. **Fail closed** — kegagalan verifikasi = tolak akses.
4. **Zero trust pada input** — semua input divalidasi & di-bind, tidak pernah
   disusun menjadi SQL/log mentah.
5. **Data minimization** — tracking & export hanya memuat data yang perlu.
6. **Auditability** — aksi sensitif selalu tercatat dan tidak dapat diubah.
7. **Secret terpisah dari kode** — env/secret manager, rotasi terdokumentasi.

---

## 2. Threat Model (STRIDE)

| Ancaman | Skenario | Kontrol |
|---|---|---|
| **Spoofing** | Pencurian access/refresh token | Access 15 menit di memori; refresh `HttpOnly`; rotasi + reuse detection; logout mencabut sesi |
| | Brute force login | Lockout 5×/15 menit; rate limit 5/menit/IP; pesan error generik |
| **Tampering** | Mengubah status/item lewat request langsung | State machine server; validasi transisi; item terkunci setelah invoice; permission per aksi |
| | Memanipulasi total invoice | Total dihitung server dari item; klien tidak mengirim total; CHECK constraint |
| | Memanipulasi stok | `FOR UPDATE`, CHECK `stock >= 0`, movement append-only |
| | Menyisipkan SQL | GORM binding/parameterized; tidak ada `Raw` dengan input; whitelist sort |
| **Repudiation** | Pengguna menyangkal perubahan status/pembayaran | `service_status_history` + `audit_logs` (actor, waktu, IP, request ID); DB role tanpa UPDATE/DELETE audit |
| **Information disclosure** | IDOR antar tenant | Scope tenant di repository; validasi relasi; RLS; test isolasi |
| | PIN perangkat bocor | AES-256-GCM; tidak pernah di list/export/log; reveal beraudit + rate limit |
| | Tracking membocorkan data | Whitelist field; tiket + 4 digit HP; rate limit; token acak |
| | Stack trace/DB error ke klien | Error handler terpusat; pesan generik 500; detail hanya di log |
| | Backup dicuri | Enkripsi backup, akses terbatas, offsite terenkripsi |
| **Denial of service** | Spam endpoint publik | Rate limit per IP; body limit; timeout; pagination wajib |
| | Upload besar | Batas 5 MB + validasi MIME; body limit 25 MB |
| **Elevation of privilege** | Teknisi mengakses endpoint kasir | Permission per route + pengecekan data di service; test RBAC |
| | Admin mengubah owner | Aturan khusus: admin tidak dapat mengelola role owner/super_admin |
| | Impersonation tanpa jejak | Super admin wajib alasan, time-boxed, audit lengkap |

---

## 3. Daftar Permission

| Modul | Permission |
|---|---|
| Dashboard | `dashboard.view` |
| Servis | `service.view`, `service.create`, `service.update`, `service.status`, `service.assign`, `service.cancel`, `service.credential.reveal`, `service.override_unpaid_pickup` |
| Diagnosis | `diagnosis.view`, `diagnosis.manage` |
| Pelanggan | `customer.view`, `customer.create`, `customer.update`, `customer.delete` |
| Sparepart | `sparepart.view`, `sparepart.create`, `sparepart.update`, `sparepart.delete` |
| Inventori | `inventory.view`, `inventory.purchase`, `inventory.adjust`, `inventory.cost.view` |
| Invoice | `invoice.view`, `invoice.create`, `invoice.update`, `invoice.void` |
| Pembayaran | `payment.view`, `payment.create`, `payment.void` |
| Garansi | `warranty.view`, `warranty.claim` |
| Laporan | `report.view`, `report.export`, `report.profit.view` |
| Pengeluaran | `expense.view`, `expense.create`, `expense.update`, `expense.delete` |
| Pengguna | `user.view`, `user.manage` |
| Outlet | `outlet.view`, `outlet.manage` |
| Pengaturan | `settings.view`, `settings.manage` |
| Audit | `audit.view` |
| Platform | `platform.tenant.view`, `platform.tenant.manage` |

### 3.1 Mapping role → permission

| Permission | Owner | Admin | Teknisi | Kasir | Super Admin |
|---|:--:|:--:|:--:|:--:|:--:|
| `dashboard.view` | ✅ | ✅ | ✅ | ✅ | — |
| `service.view` | ✅ | ✅ | ✅* | ✅ | — |
| `service.create` | ✅ | ✅ | ✅ | ✅ | — |
| `service.update` | ✅ | ✅ | ✅* | ✅ | — |
| `service.status` | ✅ | ✅ | ✅* | ✅** | — |
| `service.assign` | ✅ | ✅ | — | — | — |
| `service.cancel` | ✅ | ✅ | — | — | — |
| `service.credential.reveal` | ✅ | — | ✅* | — | — |
| `service.override_unpaid_pickup` | ✅ | — | — | — | — |
| `diagnosis.view/manage` | ✅ | ✅ | ✅* | — | — |
| `customer.*` | ✅ | ✅ | view | ✅ | — |
| `sparepart.view` | ✅ | ✅ | ✅ | ✅ | — |
| `sparepart.create/update/delete` | ✅ | ✅ | — | — | — |
| `inventory.purchase/adjust` | ✅ | ✅ | — | — | — |
| `inventory.cost.view` | ✅ | ✅ | — | — | — |
| `invoice.*` | ✅ | ✅ | — | ✅ | — |
| `payment.*` | ✅ | ✅ | — | ✅*** | — |
| `warranty.view` | ✅ | ✅ | ✅ | ✅ | — |
| `warranty.claim` | ✅ | ✅ | ✅ | ✅ | — |
| `report.view` | ✅ | ✅ | — | ✅ (harian) | — |
| `report.export` | ✅ | ✅ | — | — | — |
| `report.profit.view` | ✅ | — | — | — | — |
| `expense.*` | ✅ | ✅ | — | — | — |
| `user.view/manage` | ✅ | ✅**** | — | — | — |
| `outlet.view` | ✅ | ✅ | ✅ | ✅ | — |
| `settings.view/manage` | ✅ | ✅***** | — | — | — |
| `audit.view` | ✅ | — | — | — | — |
| `platform.tenant.*` | — | — | — | — | ✅ |

\* Hanya servis yang ditugaskan, kecuali `technician_sees_only_assigned=false`.
\** Hanya transisi `ready_for_pickup` dan `picked_up`.
\*** Tanpa `payment.void`.
\**** Tidak dapat mengelola role `owner`/`super_admin`.
\***** Tidak dapat mengubah kepemilikan tenant/plan.

---

## 4. Autentikasi & Sesi

- Password: bcrypt cost **12**, minimal 8 karakter (kombinasi huruf & angka),
  tolak password umum; tidak pernah dikembalikan lewat API.
- Super admin awal dibuat dari env; wajib ganti password saat login pertama.
- Access token: JWT **HS256** dengan secret ≥ 32 byte; klaim `iss`, `aud`, `sub`,
  `tid`, `role`, `sid`, `iat`, `exp`, `jti`; algoritma di-pin saat verifikasi.
- Refresh token: 256-bit acak, disimpan **hash SHA-256**, cookie
  `HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth`, umur 30 hari.
- Rotasi setiap refresh; reuse → cabut seluruh family + audit `token_reuse`.
- Ganti password/reset → cabut semua refresh token pengguna.
- Idle timeout klien 30 menit (UI logout otomatis), server tetap memvalidasi token.
- Login sukses/gagal, logout, lockout dicatat di audit log.
- Tidak ada "remember me" berumur panjang di MVP.

### 4.1 CSRF

Refresh token memakai `SameSite=Strict`, ditambah pemeriksaan header `Origin`
untuk semua request state-changing, dan endpoint refresh hanya menerima
`POST` dengan `Content-Type: application/json`. Endpoint API lain memakai Bearer
token sehingga tidak rentan CSRF.

---

## 5. Otorisasi & Isolasi Data

- Setiap route mendeklarasikan permission; default **deny**.
- Service memeriksa kepemilikan data (assigned technician, outlet, tenant).
- `TenantScope` wajib pada repository; tidak ada query lintas tenant kecuali
  modul platform dengan permission khusus.
- Validasi ID relasi: customer/sparepart/teknisi harus satu tenant & outlet yang
  diizinkan.
- RLS PostgreSQL (Tahap 11) sebagai lapisan terakhir:

```sql
ALTER TABLE customers ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON customers
  USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- role aplikasi BUKAN superuser / BYPASSRLS
```

- Test wajib: tenant A tidak dapat GET/PATCH/DELETE resource tenant B (IDOR),
  termasuk melalui ID yang ditebak.

---

## 6. Validasi Input & Pencegahan Injeksi

- Validasi server-side dengan `go-playground/validator` + aturan domain.
- Whitelist field yang boleh diubah (tidak memakai `ShouldBindJSON` langsung ke
  model GORM).
- GORM parameterized; `ORDER BY` memakai whitelist kolom.
- Batas panjang teks, jumlah item per request (mis. maks 100), dan ukuran body.
- HTML di-escape oleh React; jika butuh HTML (invoice), gunakan template
  server-side yang meng-escape nilai.
- Tidak ada `eval`, `dangerouslySetInnerHTML`, atau eksekusi perintah dari input.
- Normalisasi nomor telepon & email sebelum validasi/unique check.

---

## 7. Upload File

- Maks 5 MB; hanya `image/jpeg`, `image/png`, `image/webp`.
- Verifikasi **magic bytes**, bukan ekstensi/Content-Type klien.
- Nama file acak; path tidak dapat diprediksi; disimpan di luar webroot.
- Akses file melalui endpoint berautentikasi + cek tenant, atau signed URL.
- Gambar di-resize/compress (thumbnail) untuk menghemat ruang; EXIF dihapus.
- Tidak ada eksekusi file yang diunggah; direktori upload `noexec` bila memungkinkan.
- Scan malware (ClamAV) pasca-MVP.

---

## 8. Kredensial Perangkat Pelanggan

| Aspek | Kontrol |
|---|---|
| Penyimpanan | AES-256-GCM, kunci dari `APP_ENC_KEY` (32 byte base64) |
| AAD | `tenant_id‖service_order_id‖credential_type` |
| Tampilan | Write-only; tidak pernah di response list/detail/export |
| Reveal | Permission khusus, alasan wajib, rate limit 5/menit, audit, `no-store` |
| Retensi | Dihapus otomatis 30 hari setelah pickup (kecuali klaim garansi aktif) |
| Log | Nilai tidak pernah masuk log/error/analytics |
| Rotasi kunci | `credential_key_version` + job re-enkripsi |
| Larangan | Tidak meminta/menyimpan kredensial akun Google/Apple/pribadi pelanggan |

---

## 9. Privasi & Tracking Publik

- Tracking hanya memerlukan nomor tiket + 4 digit terakhir nomor HP.
- Pesan gagal selalu generik: "Data servis tidak ditemukan" (tidak membocorkan
  apakah tiket ada).
- Rate limit 10/menit/IP; token acak 128-bit; kedaluwarsa 90 hari setelah pickup.
- Field publik whitelist (lihat `06` §12); tidak ada nama lengkap, alamat,
  nomor lengkap, IMEI, PIN, foto, catatan internal, harga modal.
- Notifikasi WhatsApp hanya dikirim ke nomor pelanggan yang tercatat, dengan
  opt-out yang dihormati.
- Data pelanggan tidak dijual/dibagikan; akses dibatasi per tenant.
- Permintaan penghapusan data pelanggan: soft delete + anonimisasi data sensitif
  setelah kewajiban retensi (prosedur manual Owner + audit).

---

## 10. Audit Log

Dicatat minimal untuk:

- Login sukses/gagal, logout, lockout, token reuse.
- CRUD pengguna, perubahan role/permission/outlet.
- Pembuatan & perubahan servis, semua perubahan status.
- Reveal kredensial perangkat.
- Perubahan item, pemakaian stok, pembelian, opname.
- Invoice (issue/void), pembayaran (create/void), diskon.
- Perubahan pengaturan toko, nomor tiket, kredensial WA.
- Export laporan & backup.
- Impersonation super admin.

Isi: actor (id + nama snapshot), aksi, modul, entity, before/after (redacted),
IP, user agent, request ID, waktu. Audit log **append-only**; role DB aplikasi
tidak memiliki `UPDATE`/`DELETE` pada tabel ini. Retensi 365 hari.

---

## 11. Rate Limit & Perlindungan API

| Endpoint | Batas |
|---|---|
| `POST /auth/login` | 5/menit/IP + 5 gagal/15 menit/akun |
| `POST /auth/refresh` | 30/menit/sesi |
| `GET /public/track*` | 10/menit/IP |
| `POST /service-orders/:id/credential/reveal` | 5/menit/pengguna |
| API umum | 120/menit/pengguna |
| Upload | 20/menit/pengguna |
| Export/backup | 5/jam/pengguna |

- Limit di memori per instance (MVP), Redis saat multi-instance.
- Respons `429 RATE_LIMITED` + header `Retry-After`.
- Request timeout 30 detik; query DB timeout 10 detik; body limit 1 MB JSON.

---

## 12. Header Keamanan & Transport

```
Strict-Transport-Security: max-age=31536000; includeSubDomains
Content-Security-Policy: default-src 'self'; img-src 'self' data: blob:; ...
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: camera=(self), microphone=(), geolocation=()
Cache-Control: no-store (respons sensitif)
```

- HTTPS wajib; HTTP redirect 301.
- TLS 1.2+; cipher modern (konfigurasi Caddy/Nginx).
- CORS origin eksplisit; `Access-Control-Allow-Credentials` hanya bila perlu.
- Cookie `Secure` di production.

---

## 13. Manajemen Secret

| Secret | Lokasi | Rotasi |
|---|---|---|
| `JWT_SECRET` | env/secret manager | 90 hari (dukungan 2 kunci) |
| `APP_ENC_KEY` | env/secret manager | tahunan + re-enkripsi |
| `DB_DSN` | env | saat ganti password |
| Kredensial WA | env atau `wa_config_enc` (AES) | saat provider berubah |
| Kunci backup | secret manager | tahunan |

- Tidak ada secret di repo, image, log, atau frontend bundle.
- `.env` di-gitignore; `.env.example` hanya placeholder.
- Akses server memakai SSH key, bukan password; 2FA untuk admin server.

---

## 14. Backup & Pemulihan

```bash
# Backup harian
pg_dump -Fc -h $DB_HOST -U $DB_USER $DB_NAME > /backups/fixphone_$(date +%F).dump
tar -czf /backups/uploads_$(date +%F).tar.gz /data/uploads
age -r $BACKUP_PUBKEY -o /backups/...age /backups/...

# Restore (staging)
createdb fixphone_restore
pg_restore -d fixphone_restore /backups/fixphone_2026-11-03.dump
```

- Retensi 30 harian + 12 bulanan; salinan offsite terenkripsi.
- Uji restore kuartalan; catat RTO aktual.
- Backup tidak menyertakan kunci enkripsi; simpan kunci terpisah.
- Akses direktori backup hanya untuk proses backup & Owner.

---

## 15. Mapping OWASP Top 10 (2021)

| Risiko | Kontrol |
|---|---|
| A01 Broken Access Control | RBAC default-deny, tenant scope, RLS, test IDOR |
| A02 Cryptographic Failures | TLS, bcrypt, AES-GCM, hash refresh token, secret manager |
| A03 Injection | GORM binding, validasi, whitelist sort, escape output |
| A04 Insecure Design | Threat model, state machine, limit transaksi, review desain |
| A05 Security Misconfiguration | Header keamanan, non-root container, debug mati, CORS eksplisit |
| A06 Vulnerable Components | `govulncheck`, `npm audit`, update terjadwal, image scan |
| A07 Auth Failures | Lockout, rate limit, password policy, rotasi token |
| A08 Data Integrity Failures | Audit append-only, constraint DB, verifikasi dependency |
| A09 Logging Failures | Structured log + request ID + audit + alert |
| A10 SSRF | Tidak ada fetch URL dari pengguna; gateway WA dari konfigurasi admin |

---

## 16. Respons Insiden

1. **Deteksi** — alert error rate, login anomali, laporan pengguna.
2. **Kontainmen** — cabut token/secret terkait, nonaktifkan akun, blokir IP.
3. **Investigasi** — audit log, request ID, log server; tentukan cakupan data.
4. **Pemulihan** — perbaiki, rotasi secret, restore bila perlu.
5. **Pemberitahuan** — sesuai kewajiban hukum (UU PDP) bila data pribadi terdampak.
6. **Postmortem** — root cause, tindakan pencegahan, update dokumen.
7. **Bukti** — simpan log & kronologi; jangan timpa.

Kontak darurat dan matriks eskalasi diisi pada Tahap 12.

---

## 17. Checklist Keamanan Sebelum Go-Live

Status per Tahap 11 (12 November 2026). Item production-only tetap terbuka
sampai Tahap 12.

- [x] Semua endpoint memiliki permission & validasi.
- [x] Test IDOR lintas tenant & lintas outlet lulus.
- [x] RLS aktif dan diuji.
- [x] Tidak ada secret di repo/image/log.
- [ ] Password default diganti; super admin wajib ganti.
- [ ] HTTPS + header keamanan terverifikasi (header sudah diuji; TLS production Tahap 12).
- [x] Rate limit login & tracking terbukti.
- [x] PIN tidak muncul di response mana pun (grep + test).
- [x] Audit log mencatat aksi sensitif.
- [ ] Backup otomatis berjalan & restore teruji (restore drill lulus; jadwal otomatis Tahap 12).
- [x] Dependency scan bersih dari critical/high.
- [x] `gosec` & `npm audit` ditinjau.
- [x] Log tidak memuat data sensitif.
- [ ] Runbook insiden & kontak tersedia.
- [ ] Hak akses server (SSH, DB) minimum.
- [ ] Tidak ada akun/demo data di production.
