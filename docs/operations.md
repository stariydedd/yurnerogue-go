# Production releases

CI tests and builds each commit. Production uses the backend image tagged with
that commit's full SHA, never `latest`. The matching WASM artifact comes from
the same workflow run. Runs are serialized without cancelling an active
deployment; a run whose SHA is no longer `main` skips deployment.

The server keeps secrets in `/opt/rogue/.env` and the selected image in
`/opt/rogue/.release.env` (`BACKEND_IMAGE=ghcr.io/stariydedd/yurnerogue-backend:<sha>`).
For maintenance, use both files:

```sh
cd /opt/rogue
docker compose --env-file .env --env-file .release.env ps
docker compose --env-file .env --env-file .release.env logs --tail=100 backend
```

Deployment pulls only the selected backend image, waits for container health,
validates and reloads nginx, then checks HTTPS health, a database-backed
leaderboard read and the WASM file. A failed check fails the workflow; there is
no automatic rollback. Static upload and backend restart are not atomic, so a
short mixed-version interval is still possible. Keep API changes backwards
compatible and retain database backups independently of deployments.

To roll back, redeploy a known-good revision with its matching backend image
and web artifact. Do not delete the PostgreSQL volume or replace server secrets.

## Leaderboard protection

New clients attach a random UUID `submission_id` per run and reuse it for any
repeat attempt. The API returns `201` for a new run, `200` with the original
record for an identical replay, and `409` if that ID is reused with different
score fields. A database unique key arbitrates simultaneous requests; losing
transactions roll back their extra run. The new `run_submissions` table is
created on startup without altering or deleting existing scores. IDs are not
returned in public leaderboard records. Older clients without an ID still
work, but their submissions cannot be deduplicated.

Production nginx limits `POST /api/runs` (including a trailing slash) to an
average of 10 requests/minute per source IP with a burst allowance of 10,
plus a shared 10 requests/second ceiling with a burst allowance of 20. Excess
requests receive JSON `429` with `Retry-After: 6`; API bodies are limited to
8 KiB (`413`). Reads and static files do not consume the submission quota.
See [nginx rate-limit semantics](https://nginx.org/en/docs/http/ngx_http_limit_req_module.html).

The backend has no published port in production: keep it accessible only via
nginx. The local development stack deliberately does not include this limiter.
Limits use the socket peer IP, not untrusted forwarded headers. If adding a CDN
or another proxy, configure trusted real-IP sources first. Shared networks
(NAT) share a quota; tune the limits if legitimate players get `429`. There is
no automatic retry or durable offline outbox in the game yet, so a rejected
submission is reported, not silently retried.

These are spam and replay protections, **not anti-cheat**. A modified client
can invent scores and generate fresh UUIDs; many IPs can still fill the database
at the allowed rate. Integer bounds prevent storage overflow, not fabricated
records. Server-issued run tokens, authoritative game validation/replay,
authentication and retention/moderation are separate work. Back up PostgreSQL
regularly; do not remove historical records as part of a release.
