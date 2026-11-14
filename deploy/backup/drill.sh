#!/bin/sh
# FixPhone restore drill: restore the newest dump into a scratch database and
# verify that the schema is readable. Run weekly by the backup container. The
# scratch database must already exist and must never be the production DB.
#
# Required:
#   RESTORE_DRILL_DSN   DSN of an empty scratch database (e.g. fixphone_drill)
#   BACKUP_DSN or DB_ADMIN_DSN (source of dumps)
# Optional:
#   BACKUP_DIR          default /backups
#   BACKUP_AGE_IDENTITY path to the age private key (for encrypted dumps)
#   BACKUP_STATUS_DIR   default /backups/status
set -eu
umask 077

log() { printf '%s %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*"; }
fail() { log "ERROR: $*"; exit 1; }

DRILL_DSN="${RESTORE_DRILL_DSN:-}"
[ -n "$DRILL_DSN" ] || { log "RESTORE_DRILL_DSN tidak diisi; drill dilewati"; exit 0; }

BACKUP_DIR="${BACKUP_DIR:-/backups}"
STATUS_DIR="${BACKUP_STATUS_DIR:-/backups/status}"
MIN_TABLES="${RESTORE_DRILL_MIN_TABLES:-25}"
STARTED_AT="$(date +%s)"
mkdir -p "$STATUS_DIR"

LAST_SUCCESS="$(awk '/^fixphone_restore_drill_last_success_timestamp / {print $2}' "$STATUS_DIR/restore-drill.prom" 2>/dev/null || true)"
[ -n "$LAST_SUCCESS" ] || LAST_SUCCESS=0

write_metrics() {
    status="$1"
    duration="$2"
    tables="$3"
    now="$(date +%s)"
    success="$LAST_SUCCESS"
    [ "$status" -eq 1 ] && success="$now"
    tmp="$STATUS_DIR/.restore-drill.prom.$$"
    {
        echo "# HELP fixphone_restore_drill_last_success_timestamp Unix time of the last successful restore drill."
        echo "# TYPE fixphone_restore_drill_last_success_timestamp gauge"
        echo "fixphone_restore_drill_last_success_timestamp $success"
        echo "# HELP fixphone_restore_drill_last_status 1 when the last restore drill succeeded."
        echo "# TYPE fixphone_restore_drill_last_status gauge"
        echo "fixphone_restore_drill_last_status $status"
        echo "# HELP fixphone_restore_drill_tables Number of public tables found after restore."
        echo "# TYPE fixphone_restore_drill_tables gauge"
        echo "fixphone_restore_drill_tables $tables"
        echo "# HELP fixphone_restore_drill_duration_seconds Duration of the last restore drill."
        echo "# TYPE fixphone_restore_drill_duration_seconds gauge"
        echo "fixphone_restore_drill_duration_seconds $duration"
    } >"$tmp"
    mv "$tmp" "$STATUS_DIR/restore-drill.prom"
}

cleanup_on_error() {
    rc=$?
    if [ "$rc" -ne 0 ]; then
        log "restore drill gagal (exit $rc)"
        write_metrics 0 "$(( $(date +%s) - STARTED_AT ))" 0 || true
    fi
    exit "$rc"
}
trap cleanup_on_error EXIT

DUMP="$(ls -1t "$BACKUP_DIR"/daily/fixphone-*.dump* 2>/dev/null | head -n1 || true)"
[ -n "$DUMP" ] || fail "tidak ada dump di $BACKUP_DIR/daily"

RESTORE_FILE="$DUMP"
if [ "${DUMP##*.}" = "age" ]; then
    [ -n "${BACKUP_AGE_IDENTITY:-}" ] || fail "dump terenkripsi tetapi BACKUP_AGE_IDENTITY tidak diisi"
    command -v age >/dev/null 2>&1 || fail "age tidak tersedia"
    RESTORE_FILE="/tmp/fixphone-drill-$$.dump"
    age --decrypt --identity "$BACKUP_AGE_IDENTITY" --output "$RESTORE_FILE" "$DUMP" \
        || fail "dekripsi dump gagal"
    trap 'rm -f "$RESTORE_FILE"; cleanup_on_error' EXIT
fi

log "restore drill dari $DUMP"
pg_restore --dbname="$DRILL_DSN" --clean --if-exists --no-owner --no-privileges \
    --single-transaction "$RESTORE_FILE" || fail "pg_restore gagal"

TABLES="$(psql --dbname="$DRILL_DSN" --no-psqlrc --tuples-only --no-align \
    --command "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public'")"
TABLES="$(printf '%s' "$TABLES" | tr -d '[:space:]')"
[ "${TABLES:-0}" -ge "$MIN_TABLES" ] \
    || fail "hanya $TABLES tabel setelah restore (minimal $MIN_TABLES)"

write_metrics 1 "$(( $(date +%s) - STARTED_AT ))" "$TABLES"
trap - EXIT
[ "${RESTORE_FILE}" = "${DUMP}" ] || rm -f "$RESTORE_FILE"
log "restore drill OK: $TABLES tabel"
