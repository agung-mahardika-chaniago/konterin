# syntax=docker/dockerfile:1.7
# FixPhone backup sidecar: scheduled pg_dump, optional age encryption, offsite
# copy via rclone, weekly restore drill, and Prometheus textfile metrics.
FROM alpine:3.21

RUN apk add --no-cache \
        ca-certificates \
        tzdata \
        postgresql-client \
        age \
        rclone \
        tini \
    && mkdir -p /backups/daily /backupstatus /config

COPY entrypoint.sh backup.sh drill.sh restore.sh /usr/local/bin/
RUN chmod 0755 /usr/local/bin/*.sh

ENV BACKUP_DIR=/backups \
    BACKUP_STATUS_DIR=/backupstatus \
    UPLOAD_DIR=/data/uploads \
    TZ=Asia/Jakarta

VOLUME ["/backups"]
ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/entrypoint.sh"]
