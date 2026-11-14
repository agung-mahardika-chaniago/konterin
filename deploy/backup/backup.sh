#!/bin/sh
# FixPhone daily backup: PostgreSQL custom-format dump plus uploads archive.
# Designed for the deploy/backup image (Alpine + postgresql-client + age +
# rclone) and for manual use on a host with the same tools installed.
#
# Required: BACKUP_DSN or DB_ADMIN_DSN
# Optional:
#   BACKUP_DIR               default /backups
#   BACKUP_RETENTION_DAYS    default 14
#   BACKUP_REMOTE            rclone target, e.g. b2:fixphone-backup
#   BACKUP_AGE_RECIPIENT     age public key; encrypts every artifact
#   UPLOAD_DIR               default /data/uploads (skipped when absent)
#   BACKUP_STATUS_DIR        default /backups/status (node-exporter textfile)
set -eu
umask 077

log() { printf '%s %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*"; }
fail() { log "ERROR: $*"; exit 1; }

DSN="${BACKUP_DSN:-${DB_ADMIN_DSN:-}}"
[ -n "$DSN" ] || fail "BACKUP_DSN atau DB_ADMIN_DSN wajib diisi"

BACKUP_DIR="${BACKUP_DIR:-/backups}"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-14}"
STATUS_DIR="${BACKUP_STATUS_DIR:-/backups/status}"
UPLOAD_DIR="${UPLOAD_DIR:-/data/uploads}"
DAILY_DIR="$BACKUP_DIR/daily"
STAMP="$(date -u +%Y%m%d-T%H%M%SZ)"
HOST="$(hostname | tr -cd 'A-Za-z0-9._-')"
STARTED_AT="$(date +%s)"

mkdir -p "$DAILY_DIR" "$STATUS_DIR"

LAST_SUCCESS="$(awk '/^fixphone_backup_last_success_timestamp / {print $2}' "$STATUS_DIR/backup.prom" 2>/dev/null || true)"
[ -n "$LAST_SUCCESS" ] || LAST_SUCCESS=0

write_metrics() {
    status="$1"
    duration="$2"
    size="$3"
    now="$(date +%s)"
    success="$LAST_SUCCESS"
    [ "$status" -eq 1 ] && success="$now"
    tmp="$STATUS_DIR/.backup.prom.$$"
    {
        echo "# HELP fixphone_backup_last_success_timestamp Unix time of the last successful backup."
        echo "# TYPE fixphone_backup_last_success_timestamp gauge"
        echo "fixphone_backup_last_success_timestamp $success"
        echo "# HELP fixphone_backup_last_status 1 when the last backup succeeded."
        echo "# TYPE fixphone_backup_last_status gauge"
        echo "fixphone_backup_last_status $status"
        echo "# HELP fixphone_backup_last_attempt_timestamp Unix time of the last backup attempt."
        echo "# TYPE fixphone_backup_last_attempt_timestamp gauge"
        echo "fixphone_backup_last_attempt_timestamp $now"
        echo "# HELP fixphone_backup_duration_seconds Duration of the last backup attempt."
        echo "# TYPE fixphone_backup_duration_seconds gauge"
        echo "fixphone_backup_duration_seconds $duration"
        echo "# HELP fixphone_backup_size_bytes Total size of artifacts produced by the last backup."
        echo "# TYPE fixphone_backup_size_bytes gauge"
        echo "fixphone_backup_size_bytes $size"
    } >"$tmp"
    mv "$tmp" "$STATUS_DIR/backup.prom"
}

cleanup_on_error() {
    rc=$?
    if [ "$rc" -ne 0 ]; then
        log "backup gagal (exit $rc)"
        write_metrics 0 "$(( $(date +%s) - STARTED_AT ))" 0 || true
    fi
    exit "$rc"
}
trap cleanup_on_error EXIT

DB_FILE="$DAILY_DIR/fixphone-$STAMP.dump"
log "dump database -> $DB_FILE"
pg_dump --dbname="$DSN" --format=custom --compress=9 \
    --no-owner --no-privileges --file="$DB_FILE" || fail "pg_dump gagal"
pg_restore --list "$DB_FILE" >/dev/null 2>&1 || fail "dump tidak dapat dibaca pg_restore"
log "dump database terverifikasi"

ARTIFACTS="$DB_FILE"

if [ -d "$UPLOAD_DIR" ]; then
    UPLOAD_FILE="$DAILY_DIR/uploads-$STAMP.tar.gz"
    log "arsip uploads -> $UPLOAD_FILE"
    tar -czf "$UPLOAD_FILE" -C "$(dirname "$UPLOAD_DIR")" "$(basename "$UPLOAD_DIR")" \
        || fail "arsip uploads gagal"
    ARTIFACTS="$ARTIFACTS $UPLOAD_FILE"
fi

if [ -n "${BACKUP_AGE_RECIPIENT:-}" ]; then
    command -v age >/dev/null 2>&1 || fail "BACKUP_AGE_RECIPIENT diisi tetapi age tidak tersedia"
    ENCRYPTED=""
    for src in $ARTIFACTS; do
        age --recipient "$BACKUP_AGE_RECIPIENT" --output "$src.age" "$src" \
            || fail "enkripsi $src gagal"
        rm -f "$src"
        ENCRYPTED="$ENCRYPTED $src.age"
    done
    ARTIFACTS="$ENCRYPTED"
    log "artefak dienkripsi dengan age"
fi

TOTAL_SIZE=0
for file in $ARTIFACTS; do
    size="$(wc -c <"$file" | tr -d ' ')"
    TOTAL_SIZE=$((TOTAL_SIZE + size))
done

if [ -n "${BACKUP_REMOTE:-}" ]; then
    command -v rclone >/dev/null 2>&1 || fail "BACKUP_REMOTE diisi tetapi rclone tidak tersedia"
    RCLONE_CONFIG="${RCLONE_CONFIG:-/config/rclone.conf}"
    export RCLONE_CONFIG
    log "salin offsite -> $BACKUP_REMOTE/$HOST"
    rclone copy "$DAILY_DIR" "$BACKUP_REMOTE/$HOST" --include "*-$STAMP.*" --stats-one-line \
        || fail "rclone copy gagal"
    rclone delete "$BACKUP_REMOTE/$HOST" --min-age "${RETENTION_DAYS}d" \
        || log "PERINGATAN: retensi offsite gagal"
fi

log "hapus backup lokal lebih lama dari ${RETENTION_DAYS} hari"
find "$DAILY_DIR" -type f -mtime "+$RETENTION_DAYS" -delete 2>/dev/null || true

write_metrics 1 "$(( $(date +%s) - STARTED_AT ))" "$TOTAL_SIZE"
trap - EXIT
log "backup selesai: $TOTAL_SIZE byte"
