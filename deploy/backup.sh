#!/usr/bin/env sh
# Консистентный бэкап SQLite-базы Advisor (учитывает WAL через sqlite .backup).
# Запускать на VPS по cron. Пример (ежедневно в 3:30):
#   30 3 * * * /app/backup.sh >> /app/backups/backup.log 2>&1
set -eu

APP_DIR=${APP_DIR:-/app}
VOLUME=${VOLUME:-advisor-data} # именованный docker-том с БД
OUT_DIR=${OUT_DIR:-$APP_DIR/backups}
KEEP=${KEEP:-14} # сколько последних копий держать

mkdir -p "$OUT_DIR"
TS=$(date +%Y%m%d-%H%M%S)

# .backup через одноразовый контейнер с sqlite (рантайм-образ его не содержит),
# смонтировав именованный том с базой и папку вывода бэкапов.
docker run --rm -v "$VOLUME":/data -v "$OUT_DIR":/out alpine:3.21 sh -c \
  "apk add --no-cache sqlite >/dev/null 2>&1 && sqlite3 /data/server.db \".backup '/out/advisor-$TS.db'\""

gzip -f "$OUT_DIR/advisor-$TS.db"

# Ротация: удаляем всё старше последних $KEEP.
ls -1t "$OUT_DIR"/advisor-*.db.gz 2>/dev/null | tail -n +"$((KEEP + 1))" | xargs -r rm -f

echo "backup ok: $OUT_DIR/advisor-$TS.db.gz"
