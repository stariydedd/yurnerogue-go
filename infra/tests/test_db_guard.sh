#!/bin/sh
# Запускает scripts/db-guard.sh на одноразовом PostgreSQL и проверяет, что
# роль бэкенда rogue_app может и чего не может. Нужен Docker; не для прода.
set -eu
export MSYS_NO_PATHCONV=1   # Git Bash в Windows: пути внутри контейнеров не трогаем

# Docker в Windows ждёт пути Windows для подключаемых папок.
host_path() { if command -v cygpath >/dev/null; then cygpath -m "$1"; else printf '%s' "$1"; fi; }
here=$(cd "$(dirname "$0")" && pwd)
scripts=$(host_path "$(cd "$here/../scripts" && pwd)")
image=postgres:16.15-alpine
net=rogue-db-guard-test
db=rogue-db-guard-test-db
pw=0123456789abcdef0123456789abcdef

cleanup() {
    docker rm -f "$db" >/dev/null 2>&1 || true
    docker network rm "$net" >/dev/null 2>&1 || true
    rm -rf "$state"
}
state=$(mktemp -d)
trap cleanup EXIT

docker network create "$net" >/dev/null
docker run -d --name "$db" --network "$net" -e POSTGRES_USER=rogue \
    -e POSTGRES_PASSWORD=rogue -e POSTGRES_DB=rogue "$image" >/dev/null
for _ in $(seq 1 60); do
    docker exec "$db" pg_isready -U rogue -d rogue -h 127.0.0.1 >/dev/null 2>&1 && break
    sleep 1
done

as_rogue() { docker exec -i "$db" psql -U rogue -d rogue -v ON_ERROR_STOP=1 -qtAX "$@"; }
as_app() {
    docker run --rm -i --network "$net" -e PGPASSWORD="$pw" "$image" \
        psql -h "$db" -U rogue_app -d rogue -v ON_ERROR_STOP=1 -qtAX "$@"
}
guard() {
    docker run --rm --network "$net" -e PGHOST="$db" -e PGUSER=rogue -e PGDATABASE=rogue \
        -e PGPASSWORD=rogue -e APP_DB_PASSWORD="$1" \
        -v "$scripts:/scripts:ro" -v "$(host_path "$state"):/state" "$image" sh /scripts/db-guard.sh
}
must_fail() {
    if as_app -c "$1" >/dev/null 2>&1; then
        echo "FAIL: rogue_app was allowed: $1" >&2
        exit 1
    fi
}

# Как на проде: таблицы уже есть, их владелец rogue.
as_rogue -c "CREATE TABLE runs (id serial PRIMARY KEY, player_name text)"

guard ""
if as_rogue -c "SELECT 1 FROM pg_roles WHERE rolname = 'rogue_app'" | grep -q 1; then
    echo "FAIL: without APP_DB_PASSWORD the role must not be created" >&2
    exit 1
fi
guard "$pw"
guard "$pw"   # каждый старт запускает его снова
if guard "not-hex!" >/dev/null 2>&1; then
    echo "FAIL: a non-hex password must stop db-guard" >&2
    exit 1
fi

as_app -c "INSERT INTO runs (player_name) VALUES ('m')"
[ "$(as_app -c "SELECT count(*) FROM runs")" = 1 ]
as_app -c "UPDATE runs SET player_name = 'n'"
as_app -c "DELETE FROM runs"
as_app -c "CREATE TABLE added_later (id int)"   # create_all для новой модели
as_app -c "CREATE INDEX added_later_id ON added_later (id)"

must_fail "SELECT * FROM infra_meta"
must_fail "DROP TABLE runs"
must_fail "ALTER TABLE runs ADD COLUMN x int"
must_fail "COPY (SELECT 1) TO PROGRAM 'id'"
must_fail "SELECT pg_read_file('/etc/passwd')"
must_fail "CREATE ROLE intruder"
[ "$(as_rogue -c "SELECT rolsuper FROM pg_roles WHERE rolname = 'rogue_app'")" = f ]

# Таблица, которую rogue создаст позже, в том числе при восстановлении, доступна сразу.
as_rogue -c "CREATE TABLE restored (id int)"
as_app -c "SELECT count(*) FROM restored" >/dev/null
echo "db-guard role test passed"
