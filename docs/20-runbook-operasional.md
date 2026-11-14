# 20 — Runbook Operasional FixPhone

**Versi:** 1.0.0 · **Terakhir diperbarui:** 14 November 2026
**Berlaku untuk:** deployment production `docker-compose.prod.yml`
**Dokumen terkait:** [`07 — Keamanan`](07-keamanan.md), [`19 — Implementasi Tahap 12`](19-implementasi-tahap-12.md), [`21 — Panduan Pengguna`](21-panduan-pengguna.md)

Runbook ini adalah prosedur operasi standar (SOP) untuk tim yang menjalankan
FixPhone di production. Semua perintah diasumsikan dijalankan di server Linux
sebagai user deploy (bukan root) dari direktori aplikasi, kecuali disebut lain.

---

## 1. Ringkasan Layanan

| Aspek | Nilai |
|---|---|
| Domain | `https://<APP_DOMAIN>` (TLS otomatis Caddy) |
| Jam operasional target | 24/7, kecuali jendela pemeliharaan |
| RPO (kehilangan data maksimum) | **24 jam** — backup harian 02:15 |
| RTO (waktu pemulihan target) | **2 jam** restore lokal; **≤4 jam** bila harus unduh offsite |
| Retensi backup lokal | 14 hari (default) |
| Retensi offsite | mengikuti `BACKUP_RETENTION_DAYS` |
| Uji restore | mingguan (Minggu 03:15) bila `RESTORE_DRILL_DSN` diisi |
| Monitoring | Prometheus + Alertmanager (profil `monitoring`) |
| Status kesehatan | `GET /healthz`, `GET /readyz`, `GET /version` |

RPO 24 jam berarti insiden terburuk dapat kehilangan transaksi sejak backup
terakhir. Bila toko membutuhkan RPO lebih kecil, tambahkan backup log/streaming
(lihat §12).

### 1.1 Matriks tanggung jawab

| Peran | Tanggung jawab |
|---|---|
| **Incident commander** | Memimpin insiden, komunikasi, keputusan rollback |
| **Ops on-call** | Monitoring, triage, restart, eksekusi runbook |
| **DB owner** | Migrasi, restore, integritas data, tuning PostgreSQL |
| **Security owner** | Rotasi secret, investigasi akses, notifikasi pelanggaran |
| **Product owner** | Komunikasi pengguna, keputusan bisnis, go-live |

Satu orang boleh memegang beberapa peran pada tim kecil, tetapi **tidak boleh**
menjadi satu-satunya pemegang akses. Minimal dua orang harus dapat melakukan
deploy dan restore.

---

## 2. Akses & Prasyarat

### 2.1 Server

- Ubuntu 22.04/24.04 LTS (atau distro setara), 2 vCPU, 4 GB RAM, 40 GB SSD.
- Docker Engine 27+ dan Docker Compose v2.24+.
- Port **80** dan **443** terbuka; port lain tertutup dari internet.
- SSH hanya dengan kunci; login password root dimatikan.
- Waktu server sinkron NTP (`timedatectl`).
- DNS `A`/`AAAA` untuk `APP_DOMAIN` sudah mengarah ke server **sebelum**
  Caddy dijalankan.

### 2.2 Lokasi & izin

```bash
sudo mkdir -p /opt/fixphone
sudo chown "$USER":"$USER" /opt/fixphone
cd /opt/fixphone
git clone <repo-url> .
cp .env.production.example .env.production
chmod 600 .env.production
```

`.env.production` tidak boleh masuk Git. Password, token, dan kunci enkripsi
disimpan juga di password manager organisasi.

### 2.3 Alias Compose

Setiap perintah di runbook memakai alias berikut. Jalankan sekali per sesi
shell:

```bash
cd /opt/fixphone
dc() { docker compose --env-file .env.production -f docker-compose.prod.yml "$@"; }
```

Bila memakai PowerShell di workstation, gunakan:

```powershell
docker compose --env-file .env.production -f docker-compose.prod.yml <perintah>
```

### 2.4 Membuat secret

Jangan memakai nilai contoh. Contoh pembuatan nilai acak:

```bash
openssl rand -base64 48   # JWT_SECRET
openssl rand -base64 32   # APP_ENC_KEY (harus 32 byte setelah decode)
openssl rand -hex 24      # POSTGRES_PASSWORD
openssl rand -hex 24      # APP_DB_PASSWORD
openssl rand -base64 24   # SEED_SUPERADMIN_PASSWORD
```

`APP_DB_PASSWORD` pada `APP_DB_PASSWORD` dan `DB_DSN` **harus sama**. Gunakan
karakter aman URL (hex) agar tidak perlu percent-encoding di DSN.

---

## 3. Deploy Pertama (Go-Live)

### 3.1 Urutan eksekusi

1. **Siapkan server & DNS** (§2.1). Pastikan `dig +short <APP_DOMAIN>` sudah
   mengembalikan IP server.
2. **Isi `.env.production`**:
   - `APP_ENV=production`
   - `APP_DOMAIN`, `APP_BASE_URL=https://<domain>`, `CORS_ORIGINS=https://<domain>`
   - `ACME_EMAIL` aktif
   - `POSTGRES_PASSWORD`, `DB_ADMIN_DSN`, `APP_DB_PASSWORD`, `DB_DSN`
   - `JWT_SECRET`, `APP_ENC_KEY`
   - `SEED_SUPERADMIN_EMAIL`, `SEED_SUPERADMIN_PASSWORD`
   - `SEED_DEMO=false`
   - `TRUSTED_PROXIES` sesuai subnet Docker (default `172.16.0.0/12`)
3. **Validasi konfigurasi** tanpa menyalakan service:

   ```bash
   dc config --quiet
   ```

   Compose akan menolak start bila variabel wajib kosong.
4. **Bangun dan jalankan stack**:

   ```bash
   dc up -d --build
   ```

   Urutan otomatis: `db` sehat → `migrate` menjalankan
   `migrate up` → `seed` → `set-app-role` → `doctor` → `api` & `worker` →
   `web` → `caddy`.
5. **Pantau bootstrap**:

   ```bash
   dc ps
   dc logs -f migrate
   ```

   `migrate` harus berakhir `Exited (0)`. Bila gagal, `api` dan `worker`
   sengaja tidak dinyalakan.
6. **Verifikasi kesehatan**:

   ```bash
   curl -fsS "https://<APP_DOMAIN>/healthz"
   curl -fsS "https://<APP_DOMAIN>/readyz"
   curl -fsS "https://<APP_DOMAIN>/version"
   ```

   `/readyz` harus berisi `"status":"ready"` dan
   `"migration_version":19`. Bila `not_ready`, lihat §11.
7. **Buat tenant pertama**:

   ```bash
   dc exec api fixphone-cli create-tenant \
     --name "Nama Toko" --slug nama-toko \
     --owner-name "Nama Owner" --owner-email owner@domain.id \
     --owner-password 'PasswordKuat123' \
     --outlet-name "Outlet Utama" --outlet-code MAIN
   ```

   Simpan `tenant_id` dan `outlet_id` yang dicetak. Jangan membuat data demo.
8. **Jalankan smoke test** dari mesin di luar server:

   ```bash
   SMOKE_BASE_URL="https://<APP_DOMAIN>" \
   SMOKE_EMAIL="owner@domain.id" \
   SMOKE_PASSWORD='PasswordKuat123' \
   ./scripts/smoke-prod.sh
   ```

9. **Backup pertama** (§6.3) dan **verifikasi** hasilnya.
10. **Aktifkan monitoring** bila disiapkan (§8.1), lalu selesaikan checklist
    go-live (§13).

### 3.2 Login pertama

- Superadmin awal dibuat oleh `seed` dengan `must_change_password=true`.
- Owner tenant dibuat oleh `create-tenant`; password awal diberikan ke owner
  lewat kanal aman dan wajib diganti.
- Setelah login, ganti password dan periksa nama toko, alamat, telepon, serta
  zona waktu di **Pengaturan**.

---

## 4. Operasi Rutin

### 4.1 Harian (5–10 menit)

- [ ] `dc ps` — semua service `Up`/`healthy`; `migrate` `Exited (0)`.
- [ ] Buka dashboard; pastikan data servis dan kasir masuk.
- [ ] Periksa notifikasi/alert yang masuk sejak pemeriksaan terakhir.
- [ ] Pastikan backup 02:15 sukses (§6.3).
- [ ] Periksa kapasitas disk (`df -h /var/lib/docker`).
- [ ] Periksa antrean WhatsApp dan pengingat di halaman **Outbox** /
      **Notifikasi**; kirim ulang pesan gagal bila perlu.

### 4.2 Mingguan

- [ ] Tinjau tren error rate, latensi, dan disk (Prometheus/Grafana bila ada).
- [ ] Pastikan restore drill Minggu 03:15 sukses
      (`fixphone_restore_drill_last_success_timestamp`).
- [ ] Periksa sertifikat TLS > 14 hari.
- [ ] Periksa update image dasar dan dependency keamanan.
- [ ] Tinjau audit log untuk aksi sensitif (void, diskon, hapus, backup).
- [ ] Salin/verifikasi satu backup ke media offsite secara manual bila
      `BACKUP_REMOTE` belum dikonfigurasi.

### 4.3 Bulanan

- [ ] Uji restore penuh ke database baru (§6.5) di lingkungan staging.
- [ ] Tinjau akun aktif, role, dan akses server (least privilege).
- [ ] Tinjau kapasitas: pertumbuhan DB, uploads, dan backup.
- [ ] Terapkan patch keamanan OS dan image (dengan prosedur §5).
- [ ] Perbarui kontak darurat dan dokumen runbook bila ada perubahan.

### 4.4 Kuartalan

- [ ] Game day: simulasi API mati, DB korup, dan restore dari backup.
- [ ] Rotasi secret sesuai kebijakan (§9).
- [ ] Uji prosedur rollback aplikasi dan database di staging.
- [ ] Tinjau RPO/RTO bersama pemilik bisnis.

---

## 5. Update & Rollback

### 5.1 Prinsip

- Selalu **backup + verifikasi** sebelum update yang menyentuh skema.
- Migrasi harus kompatibel ke belakang (expand/contract). Jangan menghapus
  kolom yang masih dipakai versi lama pada rilis yang sama.
- Rollback aplikasi dan rollback database adalah dua keputusan terpisah.
- Jangan pernah menjalankan `migrate force` untuk "melewati" migrasi yang gagal.
  `force` hanya memperbaiki status dirty setelah backup dan investigasi.

### 5.2 Prosedur update rutin

```bash
cd /opt/fixphone
dc exec backup sh -lc '. /etc/backup.env; /usr/local/bin/backup.sh'   # backup pra-rilis
git fetch --tags
git checkout <tag-atau-commit>
# tinjau perubahan .env.production.example -> .env.production
dc config --quiet
dc build
dc up -d
dc logs -f migrate
dc ps
```

Setelah `migrate` selesai:

```bash
curl -fsS "https://<APP_DOMAIN>/readyz"
SMOKE_BASE_URL="https://<APP_DOMAIN>" SMOKE_EMAIL="owner@domain.id" \
  SMOKE_PASSWORD='...' ./scripts/smoke-prod.sh
```

Bila smoke gagal atau `readyz` tidak siap, lakukan rollback (§5.3) sebelum
membuka trafik lebih luas.

### 5.3 Rollback aplikasi (tanpa rollback skema)

Gunakan bila migrasi terakhir kompatibel dengan versi lama:

```bash
cd /opt/fixphone
export FIXPHONE_VERSION=<versi-lama>
dc up -d --no-deps --no-build api worker web caddy
dc ps
curl -fsS "https://<APP_DOMAIN>/readyz"
unset FIXPHONE_VERSION
```

Catatan:

- `--no-deps` mencegah service `migrate` versi lama dijalankan ulang.
- Image versi lama harus masih ada (`docker image ls | grep fixphone-backend`).
  Simpan minimal satu versi sebelumnya di server.
- Bila `readyz` melaporkan versi migrasi tidak cocok, rollback database
  diperlukan (§5.4).

### 5.4 Rollback database

**A. Migrasi reversibel.** Bila `down` tersedia dan sudah diuji:

```bash
dc run --rm --entrypoint fixphone-cli migrate migrate down --steps 1
dc run --rm --entrypoint fixphone-cli migrate migrate status
```

Lalu jalankan rollback aplikasi (§5.3).

**B. Migrasi tidak reversibel / data rusak.** Restore dump pra-rilis ke
database baru, verifikasi, lalu promosikan (§6.5). Ini pilihan yang lebih aman
daripada `down` yang menghapus kolom.

**C. Status dirty.** Setelah backup dan investigasi:

```bash
dc run --rm --entrypoint fixphone-cli migrate migrate force --version <N>
dc run --rm --entrypoint fixphone-cli migrate migrate up
dc run --rm --entrypoint fixphone-cli migrate doctor
```

### 5.5 Rollback darurat satu perintah

Bila update baru merusak layanan dan versi lama masih kompatibel:

```bash
FIXPHONE_VERSION=<versi-lama> dc up -d --no-deps --no-build api worker web caddy
```

Catat waktu, versi, alasan, dan hasil di change log.

---

## 6. Backup, Restore & Drill

### 6.1 Komponen backup

Container `backup` berjalan terus dan menjalankan:

| Waktu | Tugas | Script |
|---|---|---|
| Setiap hari 02:15 | dump DB + arsip uploads + enkripsi/offsite + retensi | `/usr/local/bin/backup.sh` |
| Setiap Minggu 03:15 | restore drill ke DB scratch | `/usr/local/bin/drill.sh` |
| Saat container start | satu backup awal | `BACKUP_ON_START=true` |

Artefak disimpan di volume `fixphone-backups` (`/backups/daily`). Metrik
ditulis ke volume `fixphone-backup-status` dan dibaca node-exporter.

Nama file: `fixphone-<host>-<UTC timestamp>.dump` dan `uploads-...tar.gz`
(opsional `.age` bila `BACKUP_AGE_RECIPIENT` diisi).

### 6.2 Verifikasi backup harian

```bash
dc ps backup
dc logs --tail=100 backup
dc exec backup ls -lh /backups/daily | tail -5
dc exec backup sh -lc '
  . /etc/backup.env
  DUMP="$(ls -1t /backups/daily/fixphone-*.dump 2>/dev/null | head -1)"
  [ -n "$DUMP" ] && pg_restore --list "$DUMP" | head
'
```

Bila `BACKUP_AGE_RECIPIENT` diisi, dump disimpan sebagai `*.dump.age`; gunakan
`pg_restore --list` setelah dekripsi (§6.5).

Metrik yang harus bernilai benar (bila profil `monitoring` aktif):

```bash
dc exec node-exporter cat /var/lib/node_exporter/textfile/backup.prom
```

- `fixphone_backup_last_status 1`
- `fixphone_backup_last_success_timestamp` < 26 jam
- `fixphone_restore_drill_last_success_timestamp` < 8 hari (bila drill aktif)

### 6.3 Backup manual (sebelum rilis/restore)

```bash
dc exec backup sh -lc '. /etc/backup.env; /usr/local/bin/backup.sh'
dc exec backup ls -lh /backups/daily | tail -3
```

Backup manual memakai jadwal dan retensi yang sama; tidak menghapus backup
lain.

### 6.4 Offsite (rclone + age)

1. Letakkan konfigurasi rclone di `deploy/backup/config/rclone.conf`
   (izin `600`, tidak masuk Git).
2. Isi `BACKUP_REMOTE` di `.env.production`, misalnya `b2:fixphone-backup`.
3. Isi `BACKUP_AGE_RECIPIENT` dengan kunci publik age. Simpan private key
   **offline** di password manager/HSM.
4. Uji:

   ```bash
   dc exec backup sh -lc '. /etc/backup.env; rclone lsd "$BACKUP_REMOTE"'
   dc exec backup sh -lc '. /etc/backup.env; /usr/local/bin/backup.sh'
   ```

5. Pastikan objek muncul di bucket dan retensi offsite berjalan.

Tanpa `BACKUP_REMOTE`, backup hanya ada di server. Risiko kehilangan total
jika server/volume hilang; minimal salin manual ke media lain setiap minggu.

### 6.5 Restore ke database baru (prosedur utama)

Jangan menimpa database produksi secara langsung. Selalu restore ke database
baru, verifikasi, lalu promosikan.

**Langkah 1 — siapkan database target.**

```bash
dc exec db sh -c 'psql -U "$POSTGRES_USER" -d postgres -v ON_ERROR_STOP=1 \
  -c "CREATE DATABASE fixphone_restore OWNER $POSTGRES_USER"'
```

**Langkah 2 — restore dump.**

```bash
dc exec backup sh -lc '
  . /etc/backup.env
  DUMP="$(ls -1t /backups/daily/fixphone-*.dump* | head -1)"
  RESTORE_DSN="postgres://postgres:<PASSWORD>@db:5432/fixphone_restore?sslmode=disable" \
    /usr/local/bin/restore.sh "$DUMP" --yes
'
```

Bila dump terenkripsi:

```bash
dc exec -e BACKUP_AGE_IDENTITY=/config/age-identity.txt backup sh -lc '...'
```

**Langkah 3 — verifikasi.**

```bash
dc exec db sh -c 'psql -U "$POSTGRES_USER" -d fixphone_restore -c "
  SELECT count(*) AS tenants FROM tenants;
  SELECT count(*) AS service_orders FROM service_orders;
  SELECT max(created_at) AS last_audit FROM audit_logs;
"'

dc exec -e DB_ADMIN_DSN="postgres://postgres:<PASSWORD>@db:5432/fixphone_restore?sslmode=disable" \
  api fixphone-cli doctor
```

`doctor` harus melaporkan 30 tabel/policy RLS, role aplikasi valid, dan tidak
ada tenant demo.

**Langkah 4 — promosikan.**

Hentikan penulisan, lalu tukar nama database:

```bash
dc stop api worker
dc exec db psql -U postgres -d postgres
```

Di prompt `psql`:

```sql
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE datname IN ('fixphone', 'fixphone_restore')
  AND pid <> pg_backend_pid();

ALTER DATABASE fixphone RENAME TO fixphone_old;
ALTER DATABASE fixphone_restore RENAME TO fixphone;
\q
```

Nama database harus sama dengan `POSTGRES_DB`/DSN. Bila `POSTGRES_USER` bukan
`postgres`, sesuaikan perintah.

**Langkah 5 — nyalakan dan verifikasi.**

```bash
dc start api worker
dc ps
curl -fsS "https://<APP_DOMAIN>/readyz"
SMOKE_BASE_URL="https://<APP_DOMAIN>" SMOKE_EMAIL="owner@domain.id" \
  SMOKE_PASSWORD='...' ./scripts/smoke-prod.sh
```

**Langkah 6 — bersihkan.**

Setelah 24 jam tanpa masalah:

```bash
dc exec db psql -U postgres -d postgres -c 'DROP DATABASE fixphone_old'
```

Simpan dump asli sampai minimal satu siklus backup baru selesai.

### 6.6 Restore drill

Aktifkan dengan membuat database scratch dan mengisi DSN:

```bash
dc exec db sh -c 'psql -U "$POSTGRES_USER" -d postgres -c "CREATE DATABASE fixphone_drill OWNER $POSTGRES_USER"'
# .env.production:
# RESTORE_DRILL_DSN=postgres://postgres:<PASSWORD>@db:5432/fixphone_drill?sslmode=disable
dc up -d backup
```

Jalankan manual:

```bash
dc exec backup sh -lc '. /etc/backup.env; /usr/local/bin/drill.sh'
dc logs --tail=50 backup
```

Drill sukses bila jumlah tabel `public` ≥ `RESTORE_DRILL_MIN_TABLES` (25) dan
metrik `fixphone_restore_drill_last_status 1`. Drill **tidak** mengubah
database produksi.

### 6.7 RPO/RTO dan keputusan restore

| Skenario | Tindakan | Perkiraan waktu |
|---|---|---|
| Data terhapus < 24 jam | Restore dump terbaru ke DB baru | 1–2 jam |
| DB korup/tidak bisa start | Restore ke DB baru, promosikan | 1–2 jam |
| Volume server hilang | Provision server baru, restore offsite | 2–4 jam |
| Salah deploy, skema kompatibel | Rollback image | 10–15 menit |
| Salah deploy, skema tidak kompatibel | Restore dump pra-rilis | 1–2 jam |

Waktu aktual bergantung ukuran DB dan kecepatan unduh offsite. Catat durasi
setiap drill untuk memperbarui RTO.

---

## 7. Insiden & Eskalasi

### 7.1 Tingkat keparahan

| Severity | Definisi | Respons awal | Update |
|---|---|---|---|
| **SEV-1** | Layanan mati total, data hilang/korup, dugaan kebocoran | 15 menit | tiap 30 menit |
| **SEV-2** | Fitur utama terganggu (login, pembayaran, tracking) | 30 menit | tiap 2 jam |
| **SEV-3** | Gangguan minor/one-user, ada workaround | 4 jam kerja | harian |

### 7.2 Alur insiden

1. **Deteksi & catat** — waktu, alert, gejala, dampak, versi.
2. **Tetapkan severity & incident commander.**
3. **Stabilkan** — jangan mengubah data tanpa backup. Opsi cepat: restart
   service, rollback image, failover DB.
4. **Batasi dampak** — nonaktifkan akun, cabut sesi, blokir IP, hentikan
   worker bila terjadi pengiriman massal.
5. **Investigasi** — log API/worker/Caddy, `audit_logs`, metrik, `docker ps`,
   `df -h`.
6. **Pulihkan** — perbaiki akar masalah, restore bila perlu, verifikasi dengan
   smoke test.
7. **Komunikasi** — beri tahu pengguna dan pemilik bisnis sesuai template.
8. **Postmortem** — dalam 3 hari kerja: kronologi, akar masalah, tindakan
   pencegahan, pemilik, tenggat. Tanpa menyalahkan individu.

### 7.3 Kontak darurat

Isi sebelum go-live dan simpan salinannya di luar server.

| Peran | Nama | Kontak | Jam |
|---|---|---|---|
| Product owner | `[isi]` | `[isi]` | `[isi]` |
| Incident commander | `[isi]` | `[isi]` | `[isi]` |
| Ops on-call | `[isi]` | `[isi]` | 24/7 |
| DB owner | `[isi]` | `[isi]` | `[isi]` |
| Security owner | `[isi]` | `[isi]` | `[isi]` |
| Penyedia server | `[isi]` | `[isi]` | 24/7 |
| Penyedia domain/DNS | `[isi]` | `[isi]` | 24/7 |

### 7.4 Dugaan kebocoran data

1. **Jangan** menghapus log atau menimpa bukti.
2. Cabut sesi: `UPDATE refresh_tokens SET revoked_at = now() WHERE revoked_at IS NULL;`
3. Rotasi `JWT_SECRET`, `APP_DB_PASSWORD`, `POSTGRES_PASSWORD`, dan token
   WhatsApp bila relevan (§9).
4. Nonaktifkan akun yang terdampak lewat UI atau SQL.
5. Simpan log, waktu, IP, dan `request_id`.
6. Evaluasi kewajiban notifikasi (UU PDP) bersama security owner.
7. Buat laporan insiden dan tindak lanjut.

---

## 8. Monitoring & Alert

### 8.1 Mengaktifkan

```bash
dc --profile monitoring up -d
dc ps
```

Prometheus, Alertmanager, node-exporter, dan blackbox-exporter hanya listen di
`127.0.0.1`. Akses dari workstation lewat SSH tunnel:

```bash
ssh -L 9090:127.0.0.1:9090 -L 9093:127.0.0.1:9093 user@server
```

Lalu buka `http://127.0.0.1:9090` dan `http://127.0.0.1:9093`.

Isi `ALERT_WEBHOOK_URL` agar alert terkirim ke Slack/Teams/relay. Tanpa webhook,
alert hanya terlihat di UI Alertmanager.

### 8.2 Metrik utama

| Metrik | Arti |
|---|---|
| `fixphone_db_up` | 1 bila API dapat ping DB |
| `fixphone_http_requests_total` | jumlah request per method/route/status |
| `fixphone_http_request_duration_seconds` | histogram latensi |
| `up{job="fixphone-api"}` | scrape API berhasil |
| `probe_success{job="blackbox-https"}` | situs dapat diakses dari luar |
| `fixphone_backup_last_status` | 1 bila backup terakhir sukses |
| `fixphone_backup_last_success_timestamp` | waktu backup sukses terakhir |
| `node_filesystem_avail_bytes` | kapasitas disk |

`/metrics` dilindungi `METRICS_TOKEN` bila diisi. Nginx mengembalikan `404`
untuk `/metrics`; Prometheus men-scrape `api:8080` di jaringan internal.

### 8.3 Respons alert

| Alert | Tindakan pertama |
|---|---|
| `FixPhoneApiDown` | `dc ps api`; `dc logs --tail=200 api`; cek DB & disk; restart `dc restart api` |
| `FixPhonePublicEndpointDown` | Cek DNS, Caddy (`dc logs caddy`), port 80/443, sertifikat |
| `FixPhoneDatabaseDown` | `dc ps db`; `dc logs db`; cek disk; jangan restart paksa bila korup |
| `FixPhoneTLSCertExpiring` | Cek log Caddy/ACME, DNS, rate limit Let's Encrypt |
| `FixPhoneApiErrorRateHigh` | Cari route/status di metrik; korelasi dengan rilis terakhir |
| `FixPhoneLatencyHigh` | Cek DB slow query, `pg_stat_activity`, disk, antrean worker |
| `FixPhoneDiskSpaceLow` | Bersihkan image/log lama; pindahkan backup; tambah disk |
| `FixPhoneDiskWillFill` | Tindakan segera; prioritaskan volume `pgdata` dan backup |
| `FixPhoneBackupFailed` | Jalankan backup manual; baca log; cek DSN, disk, rclone |
| `FixPhoneBackupStale` | Backup tidak jalan > 26 jam; cek container/cron |
| `FixPhoneRestoreDrillStale` | Cek `RESTORE_DRILL_DSN`, DB scratch, log drill |

### 8.4 Pemeriksaan manual cepat

```bash
dc ps
docker stats --no-stream
df -h
dc logs --tail=200 api
dc logs --tail=200 worker
dc logs --tail=200 caddy
dc exec db sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "
  SELECT count(*) FILTER (WHERE state = '\''active'\'') AS active,
         count(*) AS total
  FROM pg_stat_activity WHERE datname = current_database();
"'
```

---

## 9. Rotasi Secret

Lakukan saat dugaan kebocoran, pergantian personel, atau sesuai kebijakan.
Selalu backup dan lakukan di jendela pemeliharaan.

| Secret | Dampak | Prosedur |
|---|---|---|
| `POSTGRES_PASSWORD` | DB tidak bisa diakses bila salah | `ALTER ROLE` di DB → ubah `.env.production` (`POSTGRES_PASSWORD`, `DB_ADMIN_DSN`) → `dc up -d` |
| `APP_DB_PASSWORD` | API gagal konek | Ubah `APP_DB_PASSWORD` + `DB_DSN` → `dc up -d migrate` (menjalankan `set-app-role`) → `dc up -d api worker` |
| `JWT_SECRET` | Access token lama invalid (≤15 menit) | Ganti → `dc up -d api`; untuk memaksa logout semua, revoke `refresh_tokens` |
| `APP_ENC_KEY` | Kredensial perangkat tidak bisa didekripsi | **Belum ada rotasi online.** Jangan ganti tanpa re-enkripsi; simpan cadangan kunci |
| `SEED_SUPERADMIN_PASSWORD` | Hanya dipakai saat seed pertama | Reset lewat UI (Super Admin) atau prosedur pemulihan akun |
| `WA_FONNTE_TOKEN` / `WA_WABLAS_*` | Pengiriman WhatsApp gagal | Ganti di `.env.production` → `dc up -d worker` |
| `METRICS_TOKEN` | Prometheus 401 | Ganti bersamaan di `.env.production` → `dc up -d api prometheus` |
| Kunci age backup | Backup baru tidak bisa didekripsi | Buat pasangan kunci baru; simpan private key offline; backup lama tetap dengan kunci lama |

### 9.1 Contoh rotasi `APP_DB_PASSWORD`

```bash
NEWPW="$(openssl rand -hex 24)"
# 1. ubah APP_DB_PASSWORD dan DB_DSN di .env.production secara manual
# 2. terapkan ke PostgreSQL dan aktifkan login role
dc up -d migrate
# 3. pastikan doctor lulus, lalu recreate API
dc up -d --force-recreate api worker
curl -fsS "https://<APP_DOMAIN>/readyz"
```

### 9.2 Memaksa logout semua pengguna

```bash
dc exec db sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "
  UPDATE refresh_tokens SET revoked_at = now() WHERE revoked_at IS NULL;
"'
```

Rotasi `JWT_SECRET` mempercepat berakhirnya access token yang masih beredar.

---

## 10. Pemeliharaan

### 10.1 Kapasitas & disk

```bash
df -h
docker system df
du -sh /var/lib/docker/volumes/fixphone-pgdata/_data
du -sh /var/lib/docker/volumes/fixphone-backups/_data
```

Pembersihan aman:

```bash
docker image prune -f
docker builder prune -f --keep-storage 5GB
```

**Jangan** menjalankan `docker system prune --volumes` di server production.

### 10.2 Log

- Log container dibatasi 20 MB × 5 file per service.
- Log aplikasi JSON ke stdout; jangan menulis file log di container
  (filesystem read-only).
- Untuk investigasi, kumpulkan `dc logs --since 1h <service>` dan
  `request_id` dari respons error.

### 10.3 PostgreSQL

```bash
dc exec db sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "
  SELECT relname, n_live_tup, n_dead_tup, last_autovacuum
  FROM pg_stat_user_tables ORDER BY n_dead_tup DESC LIMIT 10;
"'
```

- Autovacuum dibiarkan default; pantau tabel dengan dead tuple tinggi.
- Jangan menjalankan `VACUUM FULL` saat jam sibuk.
- Koneksi maksimum: `DB_MAX_OPEN_CONNS` (25) + worker; pastikan
  `max_connections` PostgreSQL cukup.

### 10.4 Sertifikat TLS

Caddy memperbarui otomatis. Periksa:

```bash
dc logs --tail=100 caddy | grep -i -E 'certificate|tls|acme'
echo | openssl s_client -connect <APP_DOMAIN>:443 -servername <APP_DOMAIN> 2>/dev/null \
  | openssl x509 -noout -dates
```

Jangan menghapus volume `fixphone-caddy-data` tanpa alasan; volume menyimpan
sertifikat dan akun ACME.

---

## 11. Troubleshooting

| Gejala | Kemungkinan penyebab | Tindakan |
|---|---|---|
| Caddy gagal dapat sertifikat | DNS belum mengarah, port 80 tertutup, rate limit ACME | Periksa DNS/firewall, `dc logs caddy`; tunggu backoff |
| Halaman terbuka tetapi API 502 | `api` tidak sehat / mati | `dc ps api`, `dc logs api`, cek DB dan `readyz` |
| `/readyz` `not_ready` versi migrasi | `migrate` belum jalan/gagal | `dc logs migrate`; jalankan ulang `dc up -d migrate` |
| `/readyz` `dirty=true` | Migrasi terputus | Backup, investigasi, `migrate force` (§5.4C) |
| Login selalu 401 | Password salah, akun nonaktif, lockout | Cek `users.is_active`, `locked_until`, audit log |
| Semua data tenant kosong | Context tenant/RLS tidak terpasang | Cek `tenant_id` user, middleware, log `rls`; jalankan test RLS |
| Worker tidak mengirim WA | Token kosong, provider `noop`, outbox gagal | Cek pengaturan WA, `dc logs worker`, halaman Outbox |
| Upload gagal 413 | Melebihi `UPLOAD_MAX_MB`/Nginx | Sesuaikan batas dan `client_max_body_size`, restart web/api |
| Backup gagal | DSN salah, disk penuh, rclone/age error | `dc logs backup`; jalankan manual; cek `/backups` |
| Restore gagal | Dump korup, versi `pg_restore` beda, koneksi aktif | `pg_restore --list`; gunakan image backup; terminate koneksi |
| Drill basi | `RESTORE_DRILL_DSN` kosong/DB scratch tidak ada | Buat DB drill; isi DSN; recreate backup |
| Disk penuh | Image/log/backup menumpuk | Hapus image lama, rotate backup, tambah disk; **jangan** hapus `pgdata` |
| Container restart loop | Config/env salah | `dc config --quiet`, `dc logs <service>` |

### 11.1 Diagnostik aman

```bash
dc ps
dc logs --tail=300 migrate
dc logs --tail=300 api
dc exec api fixphone-cli migrate status
dc exec api fixphone-cli doctor
dc exec db sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT version, dirty FROM schema_migrations"'
```

Jangan menjalankan `migrate down`, `db-drop`, atau `DROP DATABASE` di
production tanpa backup terverifikasi dan persetujuan incident commander.

---

## 12. Peningkatan Kapasitas & Ketersediaan

Untuk pengguna bertambah:

1. **Vertikal** — tambah RAM/CPU, naikkan `DB_MAX_OPEN_CONNS` secara hati-hati.
2. **Uploads** — pindahkan ke object storage (S3-compatible) bila volume besar;
   saat ini file disimpan di volume `fixphone-uploads`.
3. **Database** — aktifkan backup PITR (WAL archiving) bila RPO < 24 jam
   dibutuhkan. Ini belum termasuk stack default.
4. **API multi-instance** — metrik saat ini process-local. Sebelum menambah
   replika, gunakan scrape per replika atau metrics store bersama; pastikan
   worker hanya satu instance agar tidak dobel kirim.
5. **Read replica** — belum didukung; perubahan besar memerlukan refactor
   routing baca.

Perubahan arsitektur harus melalui review keamanan dan pembaruan runbook ini.

---

## 13. Checklist Go-Live

### Infrastruktur

- [ ] Server memenuhi spesifikasi; waktu tersinkron NTP.
- [ ] DNS `A`/`AAAA` benar; port 80/443 terbuka; port lain tertutup.
- [ ] SSH key-only; akses root dibatasi; firewall aktif.
- [ ] Docker Engine + Compose v2 terpasang dan versi dicatat.
- [ ] `.env.production` lengkap, izin `600`, tidak masuk Git.
- [ ] Secret dibuat acak; salinan di password manager.
- [ ] `dc config --quiet` lulus.

### Aplikasi & keamanan

- [ ] `dc up -d --build` sukses; `migrate` `Exited (0)`.
- [ ] `doctor` lulus: versi 19, 30 tabel/policy RLS, role aplikasi
      `NOSUPERUSER`/`NOBYPASSRLS`, tidak ada tenant demo.
- [ ] `readyz` melaporkan `ready` dan database `up`.
- [ ] Tenant pertama dibuat; owner dapat login dan mengganti password.
- [ ] `SEED_DEMO=false`; tidak ada akun/data contoh.
- [ ] HSTS, CSP, dan cookie `Secure`/`HttpOnly`/`SameSite=Strict` terverifikasi.
- [ ] Smoke test lulus dari luar server.
- [ ] Audit log mencatat login dan aksi sensitif.

### Backup & pemulihan

- [ ] Backup manual pertama sukses dan dapat dibaca (`pg_restore --list`).
- [ ] Backup harian terjadwal; metrik `last_status=1`.
- [ ] Offsite aktif **atau** prosedur salin manual mingguan disepakati.
- [ ] Restore drill dikonfigurasi dan pernah sukses.
- [ ] Restore ke DB baru diuji minimal sekali; RTO dicatat.
- [ ] Kunci age/private key disimpan offline dan dapat diakses dua orang.

### Monitoring

- [ ] Profil `monitoring` aktif; target Prometheus `UP`.
- [ ] Webhook alert teruji (kirim alert percobaan).
- [ ] Alert API down, DB down, disk, backup, TLS terverifikasi.
- [ ] Akses Prometheus/Alertmanager hanya via localhost/SSH tunnel.
- [ ] Kontak darurat (§7.3) terisi dan dibagikan.

### Serah terima

- [ ] Admin toko dilatih: pengguna, outlet, pengaturan, backup, laporan.
- [ ] Teknisi dilatih: intake, diagnosis, item, status, kredensial, garansi.
- [ ] Kasir dilatih: invoice, pembayaran, void, cetak, tutup kasir.
- [ ] SOP harian (§4) diserahkan.
- [ ] Dokumen [`21 — Panduan Pengguna`](21-panduan-pengguna.md) dibagikan.
- [ ] Rencana komunikasi gangguan dan kanal dukungan disepakati.
- [ ] Change log dan versi (`/version`) dicatat pada hari go-live.

---

## 14. Catatan & Batasan

- Runbook ini mengasumsikan satu instance API dan satu worker. Skala horizontal
  memerlukan penyesuaian metrik dan penjadwalan worker.
- `APP_ENC_KEY` belum mendukung rotasi online; jangan menggantinya tanpa
  prosedur re-enkripsi kredensial perangkat.
- Backup default adalah logical dump; PITR belum termasuk.
- Semua contoh password pada dokumen ini adalah placeholder. Jangan pernah
  menyalinnya ke production.
