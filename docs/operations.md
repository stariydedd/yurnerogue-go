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
