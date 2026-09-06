import os
from pathlib import Path

# Должно быть выставлено до импорта app.*: движок создаётся на уровне модуля,
# и lifespan приложения выполняет create_all на нём при старте TestClient.
os.environ["DATABASE_URL"] = "sqlite://"
os.environ.setdefault("VERIFIER_PATH", str(Path(__file__).resolve().parents[2] / "build" /
                                          ("verifier.exe" if os.name == "nt" else "verifier")))

import pytest
from app.database import Base, get_db
from app.main import app
from fastapi.testclient import TestClient
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker
from sqlalchemy.pool import StaticPool


@pytest.fixture()
def client(monkeypatch):
    """TestClient с изолированной SQLite in-memory БД на каждый тест."""
    engine = create_engine(
        "sqlite://",
        connect_args={"check_same_thread": False},
        poolclass=StaticPool,
    )
    Base.metadata.create_all(bind=engine)
    monkeypatch.setattr("app.main.engine", engine)
    TestingSession = sessionmaker(bind=engine, autoflush=False, expire_on_commit=False)

    def override_get_db():
        db = TestingSession()
        try:
            yield db
        finally:
            db.close()

    app.dependency_overrides[get_db] = override_get_db
    monkeypatch.setattr("app.routers.leaderboard.secrets.randbelow", lambda n: 0)
    with TestClient(app) as c:
        def seed_run(**values):
            from app.models import Run
            with TestingSession() as db:
                row = Run(**values)
                db.add(row)
                db.commit()
                return row.id
        c.seed_run = seed_run
        yield c
    app.dependency_overrides.clear()
    engine.dispose()
