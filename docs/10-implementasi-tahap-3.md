# 10 — Implementasi Tahap 3: Auth, RBAC & User Management

**Status:** selesai · **Versi aplikasi:** 0.3.0 · **Tanggal:** 4 November 2026
**Acuan:** [`05 — Roadmap MVP`](05-roadmap-mvp.md) Tahap 3, [`06 — Kontrak API`](06-kontrak-api.md), [`07 — Keamanan`](07-keamanan.md), [`08 — UI/UX Guideline`](08-uiux-guideline.md)

---

## 1. Tujuan Tahap 3

Tahap 3 membuat aplikasi dapat dipakai oleh pengguna nyata: login aman, sesi
yang bertahan saat halaman dimuat ulang, hak akses berbasis permission, dan
pengelolaan pengguna per tenant.

Yang dihasilkan:

1. Endpoint autentikasi: login, refresh, logout, profil sendiri, dan ubah password.
2. Access token JWT HS256 berumur pendek + refresh token opak yang dirotasi.
3. Deteksi refresh token reuse yang mencabut seluruh family sesi.
4. Lockout akun, rate limit login, dan audit peristiwa autentikasi.
5. CRUD pengguna tenant, nonaktifkan, reset password, dan penugasan outlet.
6. Middleware Auth, Tenant, Permission (default deny), dan rate limiter.
7. Halaman login & ubah password responsif, guard route, `PermissionGuard`,
   pemulihan sesi otomatis, dan logout karena idle 30 menit.
8. Kontrak API Tahap 3 di [`openapi.yaml`](openapi.yaml).
9. Test integrasi RBAC, isolasi tenant, rotasi/reuse token, lockout, dan rate limit.

---

## 2. File yang Ditambahkan / Diubah

### Backend

```
backend/internal/pkg/apperror/apperror.go       # error domain + kode stabil
backend/internal/pkg/response/response.go       # envelope sukses/error + request ID
backend/internal/pkg/jwtutil/jwtutil.go         # JWT HS256 tanpa dependency
backend/internal/pkg/cryptoutil/cryptoutil.go   # random token + hash SHA-256
backend/internal/pkg/scope/scope.go             # batas tenant/outlet per request
backend/internal/pkg/validator/validator.go     # binding + pesan validasi Indonesia

backend/internal/middleware/auth.go             # verifikasi bearer + permission (default deny)
backend/internal/middleware/tenant.go           # resolusi tenant, status tenant, outlet scope
backend/internal/middleware/rate_limit.go       # sliding-window per IP + route
backend/internal/middleware/security.go         # security headers + origin check

backend/internal/modules/auth/{clock,dto,handler,routes,service}.go
backend/internal/modules/user/{dto,handler,model,repository,routes,service}.go
backend/internal/modules/outlet/{handler,model,routes,service}.go
backend/internal/modules/audit/audit.go         # penulis audit append-only

backend/internal/config/config.go               # TTL, lockout, rate limit
backend/internal/router/router.go               # wiring seluruh endpoint Tahap 3
backend/internal/pkg/version/version.go         # 0.3.0
backend/tests/integration/auth_test.go          # acceptance test Tahap 3
```

### Frontend

```
frontend/package.json                 # React 18.3, Vite 5.4, Tailwind 3.4, Zustand
frontend/vite.config.ts               # proxy /api -> 127.0.0.1:8080
frontend/tailwind.config.js           # design token navy/cyan
frontend/components.json              # kompatibel shadcn/ui
frontend/.env.example                 # VITE_API_BASE_URL, VITE_PROXY_TARGET

frontend/src/lib/auth.ts              # access token di memori + single-flight refresh
frontend/src/lib/api-client.ts        # fetch wrapper, auto-refresh, ApiError
frontend/src/lib/permissions.ts       # hasPermission, redirect awal per role
frontend/src/stores/auth-store.ts     # state sesi Zustand + bootstrap
frontend/src/features/auth/api.ts
frontend/src/features/auth/login-page.tsx
frontend/src/features/auth/change-password-page.tsx
frontend/src/features/auth/require-auth.tsx
frontend/src/features/auth/permission-guard.tsx
frontend/src/app/providers.tsx        # bootstrap sesi + idle logout
frontend/src/app/router.tsx
frontend/src/components/ui/*          # button, input, label, card, alert, spinner
frontend/src/components/shared/brand-mark.tsx
frontend/src/pages/{home,platform,forbidden,not-found}-page.tsx
frontend/src/types/{api,auth}.ts
```

---

## 3. Desain Autentikasi & Sesi

### 3.1 Access token

- JWT **HS256** dibuat tanpa library tambahan; verifikasi mengunci algoritma,
  membandingkan signature dengan `hmac.Equal`, dan memeriksa issuer/audience.
- Claims: `iss=fixphone-api`, `aud=fixphone-web`, `sub` (user ID), `tid`
  (tenant, opsional untuk super admin), `role`, `sid` (family sesi), `pv`
  (permission version), `iat`, `exp`, `jti`.
- Umur default **15 menit** (`ACCESS_TOKEN_TTL`).
- Disimpan **hanya di memori modul frontend**. Tidak ada penulisan ke
  `localStorage`/`sessionStorage`, sehingga payload XSS tidak dapat membaca
  sesi yang tersimpan.

### 3.2 Refresh token

- Token acak 256-bit (32 byte) yang dikirim sebagai cookie `fixphone_refresh`.
- Atribut cookie: `HttpOnly`, `SameSite=Strict`, `Path=/api/v1/auth`,
  `Max-Age` sesuai `REFRESH_TOKEN_TTL` (default 30 hari). `Secure` hanya aktif
  saat `APP_ENV=production`.
- Yang disimpan di database hanya **SHA-256** dari token, bukan token asli.
- Setiap refresh **merotasi** token: token lama dicabut dan dihubungkan ke
  penggantinya melalui `replaced_by_id`, di dalam satu transaksi.
- Token yang sudah dicabut lalu dipakai lagi dianggap **reuse**: seluruh token
  dalam `family_id` yang sama dicabut, kejadian `auth.token_reuse` dicatat, dan
  klien menerima `401 TOKEN_REUSED`. Family revocation di-commit sebelum error
  dikembalikan agar tidak bisa dihindari dengan retry.

### 3.3 Siklus sesi frontend

1. Saat aplikasi dimuat, `bootstrap()` memanggil `POST /auth/refresh`.
2. Bila cookie valid, access token baru disimpan di memori dan profil pengguna
   dimuat dari respons refresh; halaman tetap login setelah reload.
3. Bila access token kedaluwarsa saat memanggil API, `api-client` melakukan
   refresh **satu kali** lalu mengulang request. Permintaan refresh yang
   bersamaan digabung (single-flight) agar tidak memicu deteksi reuse sendiri.
4. Bila refresh gagal, sesi lokal dibersihkan dan pengguna diarahkan ke login.
5. Setelah 30 menit tanpa aktivitas (`mousemove`, `keydown`, `click`,
   `touchstart`, `scroll`), sesi di-logout.

### 3.4 Pencabutan sesi

| Aksi | Efek |
|---|---|
| Logout | Mencabut seluruh token pada family sesi aktif dan menghapus cookie |
| Ganti password sendiri | Menaikkan `permission_version`, mencabut **semua** refresh token pengguna |
| Reset password oleh admin | Sama seperti ganti password + `must_change_password=true` |
| Akun dinonaktifkan/dihapus | Access token lama ditolak saat `LoadAuthContext`; refresh token dicabut |
| Role diubah | `permission_version` naik sehingga access token lama langsung invalid |

`LoadAuthContext` selalu membaca ulang pengguna dari database. Access token
tidak dipercaya buta: akun nonaktif, `permission_version` yang berbeda, atau
tenant yang tidak valid akan ditolak sebelum handler berjalan.

---

## 4. Endpoint Tahap 3

Semua path berada di bawah `/api/v1`. Envelope sukses:

```json
{ "success": true, "data": { }, "request_id": "..." }
```

Envelope error:

```json
{
  "success": false,
  "error": { "code": "FORBIDDEN", "message": "Anda tidak memiliki akses untuk tindakan ini.", "details": {} },
  "request_id": "..."
}
```

### 4.1 Auth

| Method | Path | Auth | Keterangan |
|---|---|---|---|
| POST | `/auth/login` | publik + rate limit | Memasang cookie refresh; mengembalikan access token & profil |
| POST | `/auth/refresh` | cookie refresh | Rotasi token; `401 INVALID_REFRESH_TOKEN`/`TOKEN_REUSED` bila gagal |
| POST | `/auth/logout` | bearer | Mencabut family sesi dan menghapus cookie |
| GET | `/auth/me` | bearer | Profil, role, permission, dan outlet terbaru |
| PATCH | `/auth/profile` | bearer | Ubah nama, telepon, avatar (`uploads/...`) |
| POST | `/auth/change-password` | bearer | Verifikasi password lama; mencabut semua sesi |

Perilaku penting:

- Login gagal selalu mengembalikan `401 INVALID_CREDENTIALS` yang generik, baik
  email tidak ditemukan maupun password salah. Perbandingan bcrypt dummy
  menjaga waktu respons kurang lebih sama.
- Akun terkunci mengembalikan `429 ACCOUNT_LOCKED`, termasuk bila password yang
  dikirim benar.
- `must_change_password=true` tetap boleh login dan membuka `/auth/me`,
  `/auth/profile`, `/auth/change-password`, tetapi semua endpoint ber-permission
  menolak dengan `403 PASSWORD_CHANGE_REQUIRED`.
- Password baru minimal 8 karakter, maksimal 72 byte, harus mengandung huruf
  dan angka, dan berbeda dari password lama.
- `avatar_path` hanya boleh diawali `uploads/` atau `/uploads/`, tanpa `..`.

### 4.2 Pengguna, role, permission, outlet

| Method | Path | Permission | Keterangan |
|---|---|---|---|
| GET | `/users` | `user.view` | Daftar pengguna tenant; filter `q`, `role`, `status`; paginasi |
| POST | `/users` | `user.manage` | Buat pengguna; password sementara; `must_change_password=true` |
| GET | `/users/:id` | `user.view` | Detail pengguna dalam tenant |
| PATCH | `/users/:id` | `user.manage` | Ubah nama/telepon/role/status |
| DELETE | `/users/:id` | `user.manage` | Soft delete + cabut sesi |
| POST | `/users/:id/reset-password` | `user.manage` | Kembalikan password sementara satu kali |
| PUT | `/users/:id/outlets` | `user.manage` | Ganti seluruh penugasan outlet |
| GET | `/roles` | `user.view` | Role tenant + daftar permission-nya |
| GET | `/permissions` | `user.view` | Permission non-platform |
| GET | `/outlets` | `outlet.view` | Outlet sesuai scope pengguna |
| POST | `/outlets` | `outlet.manage` | Tambah outlet dalam tenant |
| PATCH | `/outlets/:id` | `outlet.manage` | Ubah outlet dalam tenant |

Aturan pengelolaan pengguna:

- `super_admin` tidak dapat dibuat atau diberikan melalui endpoint tenant.
- Owner dapat mengelola semua role tenant; admin tidak dapat membuat, mengubah,
  menghapus, mereset, atau memindahkan outlet milik **owner**.
- Pengguna tidak dapat menonaktifkan atau menghapus akunnya sendiri.
- Owner terakhir tidak dapat dinonaktifkan atau dihapus (`409 LAST_OWNER`).
- Reset password akun sendiri ditolak (`409 USE_CHANGE_PASSWORD`); gunakan
  `/auth/change-password`.
- Teknisi dan kasir wajib memiliki minimal satu outlet.
- Semua query pengguna dan outlet dibatasi `tenant_id` dari sesi, bukan dari
  input klien.

---

## 5. RBAC & Isolasi Tenant

### 5.1 Model izin

- 48 permission, 5 role sistem, 123 mapping role-permission (detail seed ada di
  [`09 — Implementasi Tahap 2`](09-implementasi-tahap-2.md)).
- Jumlah permission tenant per role: **owner 46**, **admin 42**,
  **technician 14**, **cashier 19**; `super_admin` memiliki 2 permission
  platform.
- Permission dicek di server pada setiap endpoint dengan `RequirePermission`.
  Guard frontend hanya mencegah pengguna membuka layar yang tidak relevan;
  server tetap sumber kebenaran.
- Tidak ada permission = ditolak. Tidak ada fallback ke role.

### 5.2 Batas tenant & outlet

- `middleware.Tenant` membaca status tenant: `suspended` → `403 TENANT_SUSPENDED`.
- Owner dan admin otomatis memiliki akses ke **semua outlet** tenant.
- Teknisi dan kasir hanya melihat outlet pada tabel `user_outlets`.
- Header opsional `X-Outlet-Id` harus berada dalam scope pengguna; jika tidak,
  request ditolak `403 OUTLET_FORBIDDEN`.
- Resource tenant lain diperlakukan sebagai **tidak ditemukan** (`404`), bukan
  `403`, agar keberadaannya tidak bocor.

### 5.3 Matriks yang diuji

| Skenario | Hasil |
|---|---|
| Teknisi membuka `GET /users` | `403 FORBIDDEN` |
| Teknisi membuka `GET /outlets` | `200` (outlet yang ditugaskan) |
| Admin membuka `GET /users` | `200` |
| Admin membuat pengguna role owner | `403 FORBIDDEN` |
| Admin mengubah pengguna owner | `403 FORBIDDEN` |
| Admin menonaktifkan dirinya sendiri | `409 CANNOT_DEACTIVATE_SELF` |
| Owner tenant B membaca user tenant A | `404 NOT_FOUND` |
| Owner tenant B mengubah outlet tenant A | `404 NOT_FOUND` |
| Daftar pengguna tenant B | Hanya berisi pengguna tenant B |

---

## 6. Lockout, Rate Limit & Audit

### 6.1 Lockout

- 5 kegagalan login berturut-turut dalam 15 menit mengunci akun selama 15 menit.
- Selama terkunci, login dengan password benar sekalipun ditolak
  `429 ACCOUNT_LOCKED`.
- Login berhasil mereset `failed_login_count` dan `locked_until`.
- Nilai dapat diatur: `LOGIN_MAX_ATTEMPTS`, `LOGIN_LOCK_DURATION`.

### 6.2 Rate limit

- Login dibatasi 5 request/menit/IP (`LOGIN_RATE_LIMIT`, `LOGIN_RATE_WINDOW`).
- Respons `429 RATE_LIMITED` disertai header `Retry-After`.
- Limiter disimpan di memori proses. Untuk deployment multi-instance, Tahap 12
  dapat menggantinya dengan Redis tanpa mengubah kontrak API.

### 6.3 Audit

Audit ditulis append-only ke `audit_logs` dan tidak pernah menggagalkan request
utama (`audit.BestEffort`).

| Action | Kapan |
|---|---|
| `auth.login` | Login berhasil |
| `auth.login_failed` | Email tidak ada, password salah, akun nonaktif, akun terkunci |
| `auth.lockout` | Ambang kegagalan tercapai |
| `auth.login_blocked` | Percobaan login saat masih terkunci |
| `auth.token_reuse` | Refresh token yang sudah dicabut dipakai lagi |
| `auth.logout` | Logout |
| `auth.profile_updated` | Profil sendiri diubah |
| `auth.password_changed` | Password sendiri diubah |
| `user.create` / `user.update` / `user.delete` | Perubahan pengguna oleh admin/owner |
| `user.password_reset` | Reset password oleh admin/owner |
| `user.outlets_update` | Penugasan outlet diganti |

Entry memuat `tenant_id`, `user_id`, `actor_name`, `action`, `module`,
`entity_type`, `entity_id`, data sebelum/sesudah (JSON), IP, user agent, dan
`request_id`. Password, hash, token, dan PIN tidak pernah ditulis ke audit.

---

## 7. Frontend

### 7.1 Halaman login

- Layout dua kolom di desktop (panel brand + form) dan satu kolom di mobile.
- Validasi klien: email dan password wajib; error server ditampilkan per field
  melalui `details.fields`.
- Tombol tampil/sembunyikan password, status loading, dan pesan error yang
  ramah dalam bahasa Indonesia.
- Setelah login berhasil, pengguna diarahkan ke:
  - `/change-password` bila `must_change_password`,
  - `/admin` bila `super_admin`,
  - `/` untuk role tenant.
- Bila sudah login lalu membuka `/login`, pengguna dialihkan otomatis.

### 7.2 Guard & route

| Route | Guard | Isi |
|---|---|---|
| `/login` | publik (redirect bila sudah login) | Form login |
| `/change-password` | `RequireAuth` | Ganti password wajib/sukarela |
| `/` | `RequireAuth` | Ringkasan sesi & modul berikutnya |
| `/admin` | `RequireAuth` + `PermissionGuard("platform.tenant.view")` | Area platform |
| `/403` | publik | Akses ditolak |
| `*` | publik | 404 |

`RequireAuth` menunggu status sesi selesai dipulihkan (menampilkan loader),
mengarahkan ke login bila belum terautentikasi, dan memaksa ke
`/change-password` bila password masih sementara.

### 7.3 Penanganan error & sesi

- `ApiError` membawa `status`, `code`, `details`, dan `request_id`.
- `TOKEN_EXPIRED` → refresh sekali → ulangi request.
- `401` lain → bersihkan sesi → login.
- `403 PASSWORD_CHANGE_REQUIRED` → halaman ganti password.
- Tidak ada tombol mati: aksi yang tersedia (login, logout, ganti password)
  semuanya memanggil API nyata. Modul yang belum dibangun ditandai sebagai
  pratinjau Tahap 4, bukan tombol palsu.

### 7.4 Menjalankan frontend

```powershell
cd frontend
Copy-Item .env.example .env   # opsional; default sudah memakai proxy
npm install
npm run dev                   # http://127.0.0.1:5173
```

Vite mem-proxy `/api` ke `http://127.0.0.1:8080` (dapat diubah dengan
`VITE_PROXY_TARGET`). Untuk origin API terpisah, isi `VITE_API_BASE_URL`.

Pemeriksaan:

```powershell
npm run typecheck
npm run build
```

---

## 8. Konfigurasi Environment Baru

| Variabel | Default | Aturan |
|---|---|---|
| `ACCESS_TOKEN_TTL` | `15m` | 1 menit – 24 jam |
| `REFRESH_TOKEN_TTL` | `720h` | 1 jam – 365 hari |
| `LOGIN_MAX_ATTEMPTS` | `5` | 1 – 100 |
| `LOGIN_LOCK_DURATION` | `15m` | 1 detik – 24 jam |
| `LOGIN_RATE_LIMIT` | `5` | 1 – 10000 |
| `LOGIN_RATE_WINDOW` | `1m` | 1 detik – 1 jam |

Nilai tidak valid membuat proses gagal start dengan pesan yang jelas. Semua
variabel sudah ditambahkan ke `.env.example`.

---

## 9. Hasil Pengujian

Dijalankan pada Go 1.24.5, PostgreSQL 16.15 portable, Node.js + Vite 5.4.11:

```text
go vet ./...                              OK
go build ./...                            OK
go test ./... -count=1                    OK
  internal/config                          OK
  internal/database                        OK
  tests/integration                        OK (~68s)
npm run typecheck                         OK
npm run build                             OK (224 kB JS, 17 kB CSS)
```

Test integrasi Tahap 3:

| Test | Cakupan |
|---|---|
| `TestAuthLoginSessionAndMe` | Envelope login, cookie `HttpOnly`/`Strict`/path, `expires_in=900`, owner 46 permission, 1 outlet, `/auth/me`, `401` tanpa token, tidak ada `password_hash` di respons |
| `TestAuthRefreshRotationAndReuseRevokesFamily` | Token berotasi, token lama → `401 TOKEN_REUSED`, token pengganti ikut dicabut, tidak ada token aktif tersisa, audit reuse tercatat |
| `TestAuthLogoutRevokesRefreshToken` | Cookie dibersihkan, refresh setelah logout ditolak |
| `TestAuthLockoutBlocksCorrectPasswordAndIsAudited` | 5 kegagalan mengunci akun, password benar tetap `429 ACCOUNT_LOCKED`, audit `auth.login_failed` dan `auth.lockout` |
| `TestAuthRBACAndTenantIsolation` | Matriks role, larangan admin terhadap owner, larangan nonaktifkan diri, isolasi dua tenant |
| `TestUserPasswordResetAndChangePasswordRevokeSessions` | Password sementara, `must_change_password`, access token lama invalid, refresh lama invalid, password lama ditolak |
| `TestLoginRateLimit` | Percobaan melebihi batas → `429 RATE_LIMITED` |

### Verifikasi end-to-end langsung

API dijalankan di `127.0.0.1:8080` dan Vite di `127.0.0.1:5173`:

```text
frontend_status=200
proxy_login=True role=owner perms=46
cookie_line=#HttpOnly_127.0.0.1  FALSE  /api/v1/auth  FALSE  ...  fixphone_refresh  ...
proxy_refresh=True rotated=True
proxy_me=owner@fixphone.local role=owner
html_root=<div id="root"></div>
```

Bukti tambahan:

- Login lewat proxy Vite berhasil dan memuat 46 permission owner.
- Cookie refresh memiliki prefix `#HttpOnly_` dan path `/api/v1/auth`.
- Refresh merotasi token; `/auth/me` berhasil dengan token hasil rotasi.
- Endpoint `/users`, `/roles`, dan `/outlets` merespons `200` untuk owner.
- Build produksi frontend berhasil dan HTML aplikasi tersaji.

Tidak ada browser Chrome/Edge di mesin pengembangan, sehingga verifikasi visual
manual dilakukan melalui HTTP end-to-end dan build produksi.

---

## 10. Keamanan Tahap 3

- Password memakai bcrypt cost **12**; password sementara dibuat acak dan hanya
  dikembalikan sekali pada respons reset.
- Access token hanya di memori; refresh token `HttpOnly` + `SameSite=Strict`
  dan nilainya di-hash di database.
- Rotasi refresh + reuse detection mencabut satu family sesi.
- `permission_version` memungkinkan pencabutan access token seketika.
- Semua endpoint sensitif memakai permission eksplisit; default deny.
- Tenant dan outlet berasal dari sesi/DB, bukan dari body request.
- Login generik tidak membocorkan apakah email terdaftar.
- Rate limit login dan lockout akun mengurangi brute force.
- Audit mencatat aktor, aksi, IP, user agent, dan request ID tanpa rahasia.
- Validasi input dilakukan di server; pesan error tidak menyertakan detail SQL.
- CORS memakai origin eksplisit dan cookie dikirim hanya dengan kredensial.

---

## 11. Cara Mencoba & Reset

### 11.1 Menyiapkan akun

```powershell
cd backend
go run ./cmd/cli db-setup
go run ./cmd/cli migrate up
go run ./cmd/cli seed
go run ./cmd/cli create-tenant `
  --name "Toko Demo" --slug toko-demo `
  --owner-name "Budi" --owner-email "owner@fixphone.local" --owner-password "Owner#12345" `
  --outlet-name "Outlet Utama" --outlet-code PST
go run ./cmd/api
```

### 11.2 Reset password

| Kebutuhan | Cara |
|---|---|
| Pengguna lupa password | Owner/admin memanggil `POST /api/v1/users/:id/reset-password`; password sementara ditampilkan sekali; pengguna wajib menggantinya saat login |
| Pengguna mengganti sendiri | `POST /api/v1/auth/change-password` dengan password lama dan baru |
| Semua sesi dipaksa keluar | Ganti/reset password menaikkan `permission_version` dan mencabut refresh token |
| Akun terkunci | Tunggu `LOGIN_LOCK_DURATION` (15 menit) atau kosongkan `failed_login_count`/`locked_until` lewat admin database |
| Reset total database | `go run ./cmd/cli db-drop --name fixphone_dev --yes` lalu `db-setup`, `migrate up`, `seed`, `create-tenant` |

### 11.3 Login ke frontend

1. Jalankan API (`go run ./cmd/api`) dan frontend (`npm run dev`).
2. Buka `http://127.0.0.1:5173`.
3. Masuk dengan akun owner hasil `create-tenant`.
4. Muat ulang halaman: sesi tetap aktif tanpa token di browser storage.
5. Buka menu Password untuk mencoba ganti password; semua sesi dicabut dan
   pengguna diarahkan login kembali.

---

## 12. Yang Sengaja Ditunda

| Item | Tahap |
|---|---|
| App shell (sidebar, topbar, outlet switcher) | 4 |
| Dashboard & grafik Recharts | 4 |
| Manajemen pengguna di UI (saat ini API sudah siap) | 4 |
| Rate limit terdistribusi (Redis) | 12 |
| RLS PostgreSQL | 11 |
| Enkripsi PIN perangkat & kredensial WhatsApp | 6/9 |
| PWA manifest & service worker | 4 |
| Docker image frontend/API + Nginx | 12 |

---

## 13. Definition of Done Tahap 3

- [x] Login/logout/refresh berfungsi; reload halaman tetap login.
- [x] Token tidak tersimpan di localStorage; cookie `HttpOnly`.
- [x] Setiap role hanya bisa mengakses endpoint sesuai matriks.
- [x] Reuse refresh token mencabut seluruh family.
- [x] Account lockout bekerja dan tercatat.
- [x] Integration test RBAC & isolasi tenant lulus.
- [x] `go vet ./...`, `go build ./...`, `go test ./...`, `npm run typecheck`,
      dan `npm run build` lulus.
- [x] Halaman login, ubah password, guard route, dan `PermissionGuard` tersedia.
- [x] Dokumentasi API (`docs/openapi.yaml`) dan dokumen tahap ini diperbarui.
