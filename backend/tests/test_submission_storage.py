import os
from concurrent.futures import ThreadPoolExecutor
from datetime import UTC, datetime, timedelta
from threading import Barrier
from uuid import uuid4

import pytest
from fastapi import Response
from sqlalchemy import create_engine, event, func, select
from sqlalchemy.orm import Session

from app.database import Base
from app.models import Run, RunSubmission, RankedTicket, RankedResult
from app.routers.leaderboard import submit_run
from app.schemas import RunSubmit
from tests.replay_fixture import ACTIONS


def test_additive_schema_keeps_existing_scores():
    engine = create_engine("sqlite://")
    Run.__table__.create(engine)
    RunSubmission.__table__.create(engine)
    with Session(engine) as db:
        run = Run(player_name="legacy", treasures=42, level=3)
        db.add(run)
        db.flush()
        db.add(RunSubmission(id=str(uuid4()), run_id=run.id))
        db.commit()
    Base.metadata.create_all(engine)
    Base.metadata.create_all(engine)
    with Session(engine) as db:
        assert db.scalar(select(Run)).treasures == 42
        assert db.scalar(select(func.count()).select_from(RunSubmission)) == 1
        assert db.scalar(select(func.count()).select_from(RankedResult)) == 0
    engine.dispose()


@pytest.mark.skipif(not os.getenv("TEST_POSTGRES_URL"), reason="CI PostgreSQL service required")
def test_concurrent_verified_submissions_are_atomic():
    engine = create_engine(os.environ["TEST_POSTGRES_URL"])
    Base.metadata.create_all(engine)
    key, name = str(uuid4()), uuid4().hex
    with Session(engine) as db:
        db.add(RankedTicket(id=key, seed="1", player_name=name, version="1",
                            expires_at=datetime.now(UTC) + timedelta(hours=1)))
        db.commit()
    barrier = Barrier(2)

    @event.listens_for(engine, "before_cursor_execute")
    def synchronize_inserts(conn, cursor, statement, parameters, context, executemany):
        if statement.startswith("INSERT INTO ranked_results"):
            barrier.wait(timeout=10)

    def send(index):
        with Session(engine, expire_on_commit=False) as db:
            response = Response(status_code=201)
            result = submit_run(RunSubmit(ticket=key, actions=ACTIONS), response, db)
            return response.status_code, result.id

    try:
        with ThreadPoolExecutor(max_workers=2) as pool:
            results = list(pool.map(send, range(2)))
        assert sorted(status for status, _ in results) == [200, 201]
        assert results[0][1] == results[1][1]
        with Session(engine) as db:
            assert db.scalar(select(func.count()).select_from(Run).where(Run.player_name == name)) == 1
            assert db.get(RankedResult, key) is not None
    finally:
        engine.dispose()
