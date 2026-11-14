#!/bin/sh
# Backup container entrypoint: installs cron entries from the environment and
# runs crond in the foreground. Environment variables are snapshotted to a
# root-only file because cron does not inherit the container environment.
set -eu
umask 077

: "${BACKUP_SCHEDULE:=15 2 * * *}"
: "${RESTORE_DRILL_SCHEDULE:=15 3 * * 0}"

mkdir -p /backups/daily "${BACKUP_STATUS_DIR:-/backupstatus}" /run

# Keep only variables needed by the scripts (no PATH pollution / no app secrets).
# Values are single-quoted so schedules such as "15 2 * * *" survive sourcing.
env | grep -E '^(BACKUP_|RESTORE_|DB_ADMIN_DSN|UPLOAD_DIR|TZ|PGTZ|RCLONE_)' |
    while IFS='=' read -r key value; do
        escaped=$(printf '%s' "$value" | sed "s/'/'\\\\''/g")
        printf "export %s='%s'\n" "$key" "$escaped"
    done >/etc/backup.env
chmod 600 /etc/backup.env

cat >/etc/crontabs/root <<EOF
${BACKUP_SCHEDULE} . /etc/backup.env; /usr/local/bin/backup.sh >>/proc/1/fd/1 2>&1
${RESTORE_DRILL_SCHEDULE} . /etc/backup.env; /usr/local/bin/drill.sh >>/proc/1/fd/1 2>&1
EOF
chmod 600 /etc/crontabs/root

if [ "${BACKUP_ON_START:-false}" = "true" ]; then
    echo "menjalankan backup awal"
    /usr/local/bin/backup.sh || echo "PERINGATAN: backup awal gagal"
fi

echo "backup scheduler aktif (backup: '${BACKUP_SCHEDULE}', drill: '${RESTORE_DRILL_SCHEDULE}')"
exec crond -f -l 2
