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
