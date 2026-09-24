#!/bin/bash
# Переезд production на другой сервер. Запускается с машины администратора,
# у которой есть SSH-доступ root к обоим серверам.
#
#   infra/migrate.sh OLD_IP NEW_IP
#
# Копирует /opt/rogue (секреты, state/db-id, бэкапы), статику, сертификаты и
# ключ CI, переносит базу дампом и сверяет число строк во всех таблицах.
# Старый сервер только читается: удалять его можно после смены DNS и проверки.
# Новый сервер должен быть чистым, с установленными Docker (compose) и certbot.
set -euo pipefail

if [ $# -ne 2 ]; then
    echo "usage: $0 OLD_IP NEW_IP" >&2
    exit 2
fi
OLD=root@$1
NEW=root@$2
C="docker compose --env-file .env --env-file .release.env"
ssh_old() { ssh -o BatchMode=yes "$OLD" "$@"; }
ssh_new() { ssh -o BatchMode=yes "$NEW" "$@"; }

echo "== checking servers"
ssh_new 'docker compose version >/dev/null && command -v certbot >/dev/null' ||
    { echo "install Docker (compose) and certbot on $NEW first" >&2; exit 1; }
if ssh_new 'test -e /opt/rogue'; then
    echo "$NEW already has /opt/rogue; refusing to overwrite it" >&2
    exit 1
fi

echo "== fresh backup on $OLD"
ssh_old "sh /opt/rogue/scripts/backup.sh pre-migrate"

echo "== copying files"
ssh_old 'tar -C / -czf - opt/rogue var/www/rogue etc/letsencrypt' | ssh_new 'tar -C / -xzpf -'
ssh_old 'grep yurnerogue-github-deploy /root/.ssh/authorized_keys' |
    ssh_new 'line=$(cat); mkdir -p /root/.ssh; grep -qF "$line" /root/.ssh/authorized_keys 2>/dev/null ||
        printf "%s\n" "$line" >> /root/.ssh/authorized_keys; chmod 600 /root/.ssh/authorized_keys'

echo "== moving database"
ssh_new "cd /opt/rogue && $C pull -q && $C up -d db --wait --wait-timeout 120"
ssh_old "cd /opt/rogue && $C exec -T db pg_dump -U rogue -d rogue -Fc" |
    ssh_new "cd /opt/rogue && $C exec -T db pg_restore -U rogue -d rogue --no-owner --exit-on-error"

echo "== comparing row counts"
count_sql="SELECT string_agg(format('%s=%s', table_name,
    (xpath('/row/c/text()', query_to_xml(format('SELECT count(*) AS c FROM public.%I', table_name), false, true, '')))[1]::text),
    ' ' ORDER BY table_name)
    FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE'"
old_counts=$(ssh_old "cd /opt/rogue && $C exec -T db psql -U rogue -d rogue -tAc \"$count_sql\"")
new_counts=$(ssh_new "cd /opt/rogue && $C exec -T db psql -U rogue -d rogue -tAc \"$count_sql\"")
echo "old: $old_counts"
echo "new: $new_counts"
if [ "$old_counts" != "$new_counts" ]; then
    echo "row counts differ; the new server was left with only the database running" >&2
    exit 1
fi

echo "== starting stack on $NEW (db-guard checks the database id)"
ssh_new "cd /opt/rogue && $C up -d --wait --wait-timeout 120 && $C exec -T nginx nginx -t"
curl -fsS --max-time 10 --resolve "yurnerogue.ru:443:$2" https://yurnerogue.ru/api/health >/dev/null
curl -fsS --max-time 10 --resolve "yurnerogue.ru:443:$2" "https://yurnerogue.ru/api/leaderboard?limit=1" >/dev/null

cat <<EOF
== done: $2 serves the game with the copied database.
Next steps:
  1. Point the A records of yurnerogue.ru and www.yurnerogue.ru to $2.
  2. gh secret set DEPLOY_HOST --body "$2"
  3. Update the IP in infra/nginx/rogue.conf (server_name).
  4. After DNS switches: ssh $NEW certbot renew --dry-run
  5. Runs started on $1 before the DNS switch are not copied. Remove the
     old stack only after checking the new server.
EOF
