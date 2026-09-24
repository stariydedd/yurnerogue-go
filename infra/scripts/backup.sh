#!/bin/sh
# Дамп production-базы в /opt/rogue/backups. Печатает путь к новому файлу.
# Использование: backup.sh [метка], например backup.sh pre-deploy.
# Дампы старше 14 дней удаляются; копии вне сервера хранит workflow backup.yml.
set -eu

cd "$(dirname "$0")/.."
umask 077
mkdir -p backups

name="backups/rogue-$(date -u +%Y%m%dT%H%M%SZ)${1:+-$1}.dump"
docker compose --env-file .env --env-file .release.env exec -T db \
    pg_dump -U rogue -d rogue -Fc > "$name.tmp"
# Пустой или обрезанный дамп не должен выглядеть как успешный бэкап.
docker compose --env-file .env --env-file .release.env exec -T db \
    pg_restore --list < "$name.tmp" > /dev/null
mv "$name.tmp" "$name"

find backups -name 'rogue-*.dump' -mtime +14 -delete
echo "$(pwd)/$name"
