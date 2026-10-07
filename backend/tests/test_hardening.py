from app.database import Base, get_db
from app.main import app
from app.models import RankedTicket, Run
from fastapi.testclient import TestClient
import pytest
from sqlalchemy import create_engine, inspect
from sqlalchemy.pool import StaticPool


def started_name(client, name):
    response = client.post("/api/runs/start", json={"player_name": name, "version": "2"})
    assert response.status_code == 201
    dependency = app.dependency_overrides[get_db]()
    with next(dependency) as db:
        stored = db.get(RankedTicket, response.json()["ticket"]).player_name
    dependency.close()
    return stored


@pytest.mark.parametrize("sent,stored", [
    ("Juggernaut", "Juggernaut"),
    ("  Сайлент  ", "Сайлент"),
    ("line\nbreak\ttab", "linebreaktab"),
    ("zero\u200bwidth\u200d", "zerowidth"),
    ("\u202egnirts desrever", "gnirts desrever"),
    ("sixteen_chars_ok_and_more", "sixteen_chars_ok"),
    ("fifteen chars  !", "fifteen chars  !"),
    ("   \u200b\n", "anonymous"),
])
def test_player_names_are_cleaned_like_the_game_does(client, sent, stored):
    assert started_name(client, sent) == stored


def test_api_refuses_other_origins(client):
    preflight = client.options("/api/runs/start", headers={
        "Origin": "https://evil.example", "Access-Control-Request-Method": "POST",
    })
    assert "access-control-allow-origin" not in preflight.headers
    response = client.get("/api/leaderboard", headers={"Origin": "https://evil.example"})
    assert response.status_code == 200
    assert "access-control-allow-origin" not in response.headers


def test_leaderboard_index_is_added_to_an_existing_table(monkeypatch):
    engine = create_engine("sqlite://", connect_args={"check_same_thread": False}, poolclass=StaticPool)
    Base.metadata.create_all(bind=engine)
    for index in Run.__table__.indexes:
        index.drop(bind=engine)  # a database made before the index existed
    monkeypatch.setattr("app.main.engine", engine)
    with TestClient(app):
        pass
    assert "runs_leaderboard_order" in {index["name"] for index in inspect(engine).get_indexes("runs")}
    with TestClient(app):  # and a restart does not trip over it
        pass
    engine.dispose()
