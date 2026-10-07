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

# Роль приложения: читает и пишет таблицы игры, может создавать новые, но не
# суперпользователь (нет COPY ... PROGRAM, чтения файлов сервера, чужих таблиц
# и infra_meta). Дамп базы не содержит ролей, поэтому роль и права выдаются
# здесь при каждом старте: после восстановления или переезда тоже. Без
# APP_DB_PASSWORD в .env шаг пропускается, backend остаётся на суперпользователе.
if [ -n "${APP_DB_PASSWORD:-}" ]; then
    case "$APP_DB_PASSWORD" in
        *[!0-9a-f]*)
            echo "db-guard: APP_DB_PASSWORD must be lowercase hex (openssl rand -hex 24)" >&2
            exit 1 ;;
    esac
    psql -v ON_ERROR_STOP=1 -qX -v pw="$APP_DB_PASSWORD" <<'SQL'
SELECT format('CREATE ROLE rogue_app LOGIN PASSWORD %L', :'pw')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'rogue_app') \gexec
ALTER ROLE rogue_app WITH LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'pw';
SELECT format('GRANT CONNECT ON DATABASE %I TO rogue_app', current_database()) \gexec
GRANT USAGE, CREATE ON SCHEMA public TO rogue_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO rogue_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO rogue_app;
REVOKE ALL ON infra_meta FROM rogue_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO rogue_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO rogue_app;
SQL
    echo "db-guard: role rogue_app is up to date"
fi

echo "db-guard: ok, db_id $db_id"
