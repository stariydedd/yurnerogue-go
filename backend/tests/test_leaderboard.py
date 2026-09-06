from datetime import UTC, datetime, timedelta
from uuid import uuid4

import pytest
from app.database import get_db
from app.main import app
from app.models import RankedTicket
from tests.replay_fixture import ACTIONS


def start(client, name="tester"):
    response = client.post("/api/runs/start", json={"player_name": name, "version": "1"})
    assert response.status_code == 201
    assert response.json()["seed"] == "1"
    return response.json()["ticket"]


def submit(client, ticket, actions=ACTIONS, **extra):
    return client.post("/api/runs", json={"ticket": ticket, "actions": actions, **extra})


def test_health(client):
    assert client.get("/api/health").json() == {"status": "ok"}


def test_server_recomputes_score(client):
    response = submit(client, start(client))
    assert response.status_code == 201
    body = response.json()
    assert body["player_name"] == "tester"
    assert body["treasures"] == 278
    assert body["enemies_killed"] == 6
    assert body["attacks_made"] == 45
    assert body["hits_taken"] == 20
    assert body["tiles_moved"] == 217
    assert body["verified"] is True
    assert "ticket" not in body and "actions" not in body


@pytest.mark.parametrize("payload", [
    {"treasures": 9999, "level": 21},
    {"submission_id": str(uuid4()), "player_name": "cheater", "treasures": 9999, "level": 21},
])
def test_legacy_score_writes_are_closed(client, payload):
    assert client.post("/api/runs", json=payload).status_code == 422
    assert client.get("/api/leaderboard").json() == []


@pytest.mark.parametrize("extra", [{"treasures": 9999}, {"level": 21},
                                  {"player_name": "forged"}, {"verified": True}, {"seed": "1"}])
def test_client_cannot_override_verified_fields(client, extra):
    assert submit(client, start(client), **extra).status_code == 422
    assert client.get("/api/leaderboard").json() == []


def test_ticket_is_required_and_server_issued(client):
    assert submit(client, str(uuid4())).status_code == 404
    assert submit(client, "bad").status_code == 422
    assert client.post("/api/runs/start", json={"version": "1", "seed": "1"}).status_code == 422


@pytest.mark.parametrize("actions", ["w", "z", "h9", "h", "d100", "", "w" * 60001, ACTIONS + "w"],
                         ids=["unfinished", "awake-wait", "missing-item", "truncated", "bad-code", "empty", "too-long", "after-death"])
def test_invalid_or_unfinished_replays_are_rejected(client, actions):
    assert submit(client, start(client), actions).status_code == 422
    assert client.get("/api/leaderboard").json() == []


def test_idempotent_replay_and_conflict(client):
    ticket = start(client)
    first = submit(client, ticket)
    again = submit(client, ticket)
    assert first.status_code == 201 and again.status_code == 200
    assert first.json() == again.json()
    assert submit(client, ticket, "w").status_code == 409
    assert len(client.get("/api/leaderboard").json()) == 1


def test_same_name_and_score_allowed_for_distinct_runs(client):
    for _ in range(2):
        assert submit(client, start(client, "same")).status_code == 201
    assert len(client.get("/api/leaderboard").json()) == 2


def test_legacy_scores_are_grandfathered_as_trusted(client):
    old_id = client.seed_run(player_name="legacy", treasures=1000, level=9)
    submit(client, start(client))
    rows = client.get("/api/leaderboard").json()
    assert [(r["id"], r["verified"]) for r in rows][:1] == [(old_id, True)]
    assert rows[0]["player_name"] == "legacy" and rows[0]["treasures"] == 1000
    assert rows[1]["verified"] is True
    assert len(client.get("/api/leaderboard?limit=1").json()) == 1


def test_leaderboard_sorts_legacy_records_by_score_then_level(client):
    for gold, level in [(100, 9), (300, 1), (100, 21)]:
        client.seed_run(player_name="legacy", treasures=gold, level=level)
    rows = client.get("/api/leaderboard").json()
    assert [(r["treasures"], r["level"]) for r in rows] == [(300, 1), (100, 21), (100, 9)]


@pytest.mark.parametrize("field,value,status", [
    ("expires_at", datetime.now(UTC) - timedelta(seconds=1), 410),
    ("version", "old", 409),
])
def test_expired_or_incompatible_tickets_rejected(client, field, value, status):
    ticket = start(client)
    dependency = app.dependency_overrides[get_db]()
    with next(dependency) as db:
        row = db.get(RankedTicket, ticket)
        setattr(row, field, value)
        db.commit()
    dependency.close()
    assert submit(client, ticket).status_code == status


def test_start_validates_version_and_name(client):
    assert client.post("/api/runs/start", json={"version": "old"}).status_code == 409
    assert client.post("/api/runs/start", json={"version": "1", "player_name": "x" * 33}).status_code == 422
    ticket = client.post("/api/runs/start", json={"version": "1"}).json()["ticket"]
    assert submit(client, ticket).json()["player_name"] == "anonymous"


def test_verifier_failure_is_closed(client, monkeypatch):
    ticket = start(client)
    monkeypatch.setattr("app.verifier.VERIFIER_PATH", "/missing-verifier")
    assert submit(client, ticket).status_code == 503
    assert client.get("/api/leaderboard").json() == []


def test_verifier_busy_is_closed_and_recoverable(client):
    from app.verifier import _slots
    ticket = start(client)
    with _slots, _slots:
        assert submit(client, ticket).status_code == 503
    assert submit(client, ticket).status_code == 201


def test_verifier_timeout_is_closed_and_releases_slot(client, monkeypatch):
    import subprocess
    ticket = start(client)
    with monkeypatch.context() as patch:
        def timeout(*args, **kwargs):
            raise subprocess.TimeoutExpired("verifier", 3)
        patch.setattr("app.verifier.subprocess.run", timeout)
        assert submit(client, ticket).status_code == 422
        assert client.get("/api/leaderboard").json() == []
    assert submit(client, ticket).status_code == 201
