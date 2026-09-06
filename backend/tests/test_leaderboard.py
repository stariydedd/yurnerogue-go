from uuid import uuid4

import pytest


def _run(treasures=100, level=3, **kw):
    payload = {"player_name": "tester", "treasures": treasures, "level": level}
    payload.update(kw)
    return payload


def test_health(client):
    resp = client.get("/api/health")
    assert resp.status_code == 200
    assert resp.json() == {"status": "ok"}


def test_submit_run_returns_created_record(client):
    resp = client.post("/api/runs", json=_run(treasures=250, level=5, enemies_killed=7))
    assert resp.status_code == 201
    body = resp.json()
    assert body["player_name"] == "tester"
    assert body["treasures"] == 250
    assert body["enemies_killed"] == 7
    assert body["id"] > 0
    assert body["created_at"]


def test_leaderboard_sorted_by_treasures_then_level(client):
    client.post("/api/runs", json=_run(treasures=100, level=9))
    client.post("/api/runs", json=_run(treasures=300, level=1))
    client.post("/api/runs", json=_run(treasures=100, level=21))

    resp = client.get("/api/leaderboard")
    assert resp.status_code == 200
    rows = resp.json()
    assert [(r["treasures"], r["level"]) for r in rows] == [(300, 1), (100, 21), (100, 9)]


def test_leaderboard_respects_limit(client):
    for i in range(5):
        client.post("/api/runs", json=_run(treasures=i))
    resp = client.get("/api/leaderboard", params={"limit": 3})
    assert len(resp.json()) == 3


def test_submit_rejects_negative_treasures(client):
    resp = client.post("/api/runs", json=_run(treasures=-5))
    assert resp.status_code == 422


def test_submit_rejects_level_out_of_range(client):
    resp = client.post("/api/runs", json=_run(level=22))
    assert resp.status_code == 422


def test_submit_rejects_too_long_name(client):
    resp = client.post("/api/runs", json=_run(player_name="x" * 33))
    assert resp.status_code == 422


def test_default_player_name_is_anonymous(client):
    resp = client.post("/api/runs", json={"treasures": 10, "level": 1})
    assert resp.status_code == 201
    assert resp.json()["player_name"] == "anonymous"


def test_replay_returns_original_record_without_duplicate(client):
    payload = _run(submission_id=str(uuid4()))
    first = client.post("/api/runs", json=payload)
    again = client.post("/api/runs", json=payload)
    assert first.status_code == 201
    assert again.status_code == 200
    assert again.json() == first.json()
    assert len(client.get("/api/leaderboard").json()) == 1


def test_reusing_key_with_different_score_is_conflict(client):
    payload = _run(submission_id=str(uuid4()))
    first = client.post("/api/runs", json=payload)
    payload["treasures"] += 1
    assert client.post("/api/runs", json=payload).status_code == 409
    assert client.get("/api/leaderboard").json() == [first.json()]


def test_identical_scores_from_distinct_runs_are_allowed(client):
    for _ in range(2):
        assert client.post("/api/runs", json=_run(submission_id=str(uuid4()))).status_code == 201
    assert len(client.get("/api/leaderboard").json()) == 2


def test_replay_normalizes_uuid_and_default_fields(client):
    key = str(uuid4())
    first = client.post("/api/runs", json={"treasures": 10, "level": 1, "submission_id": key})
    again = client.post("/api/runs", json={**first.json(), "submission_id": key.upper()})
    assert again.status_code == 200
    assert again.json() == first.json()


def test_malformed_submission_id_is_rejected(client):
    assert client.post("/api/runs", json=_run(submission_id="invalid")).status_code == 422


@pytest.mark.parametrize("field", [
    "treasures", "enemies_killed", "food_used", "elixirs_used", "scrolls_read",
    "attacks_made", "hits_taken", "tiles_moved",
])
def test_counter_bounds_match_postgresql(client, field):
    assert client.post("/api/runs", json=_run(**{field: 2**31})).status_code == 422
    assert client.post("/api/runs", json=_run(**{field: -1})).status_code == 422
    assert client.post("/api/runs", json=_run(**{field: 2**31 - 1})).status_code == 201
