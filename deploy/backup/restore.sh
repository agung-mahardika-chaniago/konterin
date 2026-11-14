#!/bin/sh
# FixPhone manual restore. Restores a custom-format dump into an explicit
# target database. Production should normally be restored into a new database,
# verified, and only then promoted; see docs/20-runbook-operasional.md.
#
# Usage:
#   restore.sh <dump-file> [target-dsn] --yes
#   RESTORE_DSN=postgres://... restore.sh /backups/daily/fixphone-....dump --yes
#
# Optional: BACKUP_AGE_IDENTITY for *.dump.age files.
set -eu
umask 077

log() { printf '%s %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*"; }
fail() { log "ERROR: $*"; exit 1; }
usage() {
    echo "usage: restore.sh <dump-file> [target-dsn] --yes"
    echo "       target-dsn dapat diganti dengan RESTORE_DSN"
    exit 2
}

DUMP=""
TARGET="${RESTORE_DSN:-}"
CONFIRMED=0
for arg in "$@"; do
    case "$arg" in
        --yes) CONFIRMED=1 ;;
        --help|-h) usage ;;
        *)
            if [ -z "$DUMP" ]; then DUMP="$arg"; else TARGET="$arg"; fi
            ;;
    esac
done

[ -n "$DUMP" ] || usage
[ -f "$DUMP" ] || fail "dump tidak ditemukan: $DUMP"
[ -n "$TARGET" ] || fail "target DSN wajib diisi (argumen kedua atau RESTORE_DSN)"
[ "$CONFIRMED" -eq 1 ] || fail "tambahkan --yes untuk menimpa database target"

RESTORE_FILE="$DUMP"
if [ "${DUMP##*.}" = "age" ]; then
    [ -n "${BACKUP_AGE_IDENTITY:-}" ] || fail "dump terenkripsi tetapi BACKUP_AGE_IDENTITY tidak diisi"
    RESTORE_FILE="/tmp/fixphone-restore-$$.dump"
    age --decrypt --identity "$BACKUP_AGE_IDENTITY" --output "$RESTORE_FILE" "$DUMP" \
        || fail "dekripsi dump gagal"
    trap 'rm -f "$RESTORE_FILE"' EXIT
fi

log "memeriksa isi dump"
pg_restore --list "$RESTORE_FILE" >/dev/null 2>&1 || fail "dump tidak dapat dibaca"

log "restore ke database target (objek lama akan diganti)"
pg_restore --dbname="$TARGET" --clean --if-exists --no-owner --no-privileges \
    --single-transaction "$RESTORE_FILE" || fail "pg_restore gagal"

TABLES="$(psql --dbname="$TARGET" --no-psqlrc --tuples-only --no-align \
    --command "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public'")"
log "restore selesai: $(printf '%s' "$TABLES" | tr -d '[:space:]') tabel public"
log "jalankan 'fixphone-cli doctor' dan smoke test sebelum membuka trafik"
