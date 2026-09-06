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

## Ranked replay verification

1. Before playing, the client requests `POST /api/runs/start` with
   `{"player_name":"name","version":"1"}`.
2. The server chooses the seed and returns an unpredictable one-run ticket,
   a decimal-string seed and the rules version. The ticket expires in 24 hours.
   Its name and seed cannot be changed by the final submission.
3. The client uses a session-local RNG and records legal input actions.
4. `POST /api/runs` accepts only `{"ticket":"uuid","actions":"..."}`. The Go
   verifier reproduces the entire run with the same domain package as WASM.
   Only a terminal run (death or victory) can be saved; all score fields are
   calculated on the server. Client-supplied score fields are rejected.
5. The response is `201` for a new verified record, `200` for an exact replay,
   or `409` if an already used ticket receives another action log. A unique
   database key prevents concurrent duplicate scores. Completed retries still
   work after ticket expiry; an unsubmitted expired ticket returns `410`.

The new `ranked_tickets` and `ranked_results` tables are additive. Existing
scores and the old `run_submissions` table are **not deleted or rewritten**.
The owner has grandfathered existing records as trusted: the API returns
`verified: true` for both historical and newly replayed scores. The game shows
no verification column or legend. This is an acceptance policy, not retroactive
proof of old gameplay: no replay hashes or tickets are fabricated for legacy
records. New writes still require replay verification. Tickets and action logs
are not public leaderboard fields; only the log hash is stored.

The old arbitrary-score POST protocol is intentionally closed, even for clients
with a `submission_id`. Players must reload the page before starting a ranked
run. An unavailable/incompatible start falls back to a clearly announced
practice run, which is not submitted. There is no mid-run network requirement
or durable offline outbox. A finished ranked run can retry with R (RUN on touch)
while its end screen remains open, using the same ticket and journal.

### Resource limits

Both start and finish POST endpoints share nginx's average 10 requests/minute
per socket peer IP with a burst allowance of 10, plus 10 requests/second
globally with a burst allowance of 20. Excess requests return JSON `429` and
`Retry-After: 6`. Reads and static files do not consume this quota.
API bodies are capped at 64 KiB, journals at 60,000 ASCII bytes and simulated
turns at 100,000. Over-limit play can continue locally but cannot be ranked.

Verification uses at most two subprocesses per API worker, each with one Go
CPU thread, a 64 MiB soft Go memory target and a 3-second wall-clock timeout.
Subprocess input is bounded; there is no shell or execution of supplied code.
Invalid/unfinished/over-budget replay returns `422`; busy/broken verification
returns `503`. Neither creates a record. Startup fails if the verifier binary
is missing. These limits may need tuning for long legitimate runs or slower
hardware; the memory target is not an OS hard limit.

Keep the production backend private behind nginx. Local development exposes
the backend directly and intentionally has no proxy rate limiter. If adding
a CDN, configure trusted real-IP sources first; forwarded headers alone do not
change the quota key. Players behind the same NAT share a quota.

### Rules changes, testing and remaining risks

The Docker build context is the repository root. The backend multi-stage image
compiles `cmd/verifier` and ships it alongside FastAPI; no extra service or
production secret is required. For backend tests, first build
`go build -o build/verifier ./cmd/verifier` (use `build/verifier.exe` on Windows).
For running FastAPI outside Docker, set `VERIFIER_PATH` to its absolute path.

Increment `domain.RulesVersion` for simulation changes, including changes in
RNG call order. Refresh the golden replay fixtures deliberately. WASM and native
Go tests check the same golden result; API tests invoke the real verifier.
An in-flight run from a different rules version cannot be verified after a
release. Supporting old engines for the ticket lifetime is separate work.
Do not roll back to an older arbitrary-score API without understanding that it
reopens unverified writes.

This verifies **rule-consistent outcomes, not human play**. The public seed and
client simulation allow prediction, bots, route search and replay assistance.
Names are not authenticated identities. Distributed clients can still fill
storage at the allowed rate; ticket and result retention/moderation remain
operational work. Legacy scores are accepted by owner decision rather than
replay evidence and can still lead the shared ranking. Back up PostgreSQL and monitor verification
latency, rejection rates and table sizes; this release removes no records.
