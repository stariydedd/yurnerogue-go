#!/bin/sh
# Не даёт backend стартовать на пустой или чужой базе.
#
# Backend создаёт таблицы сам (create_all), поэтому без этой проверки сервер
# с потерянной базой поднимается «рабочим» и с пустой таблицей рекордов.
# Идентификатор базы хранится и в самой базе (infra_meta), и в файле
# /opt/rogue/state/db-id. При переезде оба копируются вместе; если скопировали
# только файлы, а базу нет, идентификаторы не совпадут и backend не запустится.
set -eu

q() { psql -v ON_ERROR_STOP=1 -tAq -c "$1"; }

# Ничего не создаём до проверки: пустая база должна остаться пустой, чтобы
# в неё можно было восстановить дамп без конфликтов.
db_id=""
if [ "$(q "SELECT to_regclass('public.infra_meta') IS NOT NULL")" = t ]; then
    db_id=$(q "SELECT value FROM infra_meta WHERE key = 'db_id'")
fi
file_id=$(cat /state/db-id 2>/dev/null || true)

if [ -z "$db_id" ]; then
    if [ -n "$file_id" ]; then
        echo "db-guard: database has no db_id, server expects $file_id" >&2
        echo "db-guard: restore the database (docs/operations.md), or delete" >&2
        echo "db-guard: /opt/rogue/state/db-id to start an intentionally empty one" >&2
        exit 1
    fi
    q "CREATE TABLE IF NOT EXISTS infra_meta (key text PRIMARY KEY, value text NOT NULL)"
    db_id=$(q "INSERT INTO infra_meta VALUES ('db_id', gen_random_uuid()::text) RETURNING value")
    echo "db-guard: database had no id, assigned db_id $db_id"
fi

if [ -z "$file_id" ]; then
    printf '%s\n' "$db_id" > /state/db-id
    echo "db-guard: saved db_id to state/db-id"
elif [ "$file_id" != "$db_id" ]; then
    echo "db-guard: database db_id $db_id does not match server state/db-id $file_id" >&2
    exit 1
fi

echo "db-guard: ok, db_id $db_id"
