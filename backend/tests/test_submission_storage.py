import os
from concurrent.futures import ThreadPoolExecutor
from threading import Barrier
from uuid import uuid4

import pytest
from fastapi import HTTPException, Response
from sqlalchemy import create_engine, event, func, select
from sqlalchemy.orm import Session

from app.database import Base
from app.models import Run, RunSubmission
from app.routers.leaderboard import submit_run
from app.schemas import RunSubmit


def test_additive_schema_keeps_existing_scores():
    engine = create_engine("sqlite://")
    Run.__table__.create(engine)
    with Session(engine) as db:
        db.add(Run(player_name="legacy", treasures=42, level=3))
        db.commit()
    Base.metadata.create_all(engine)
    Base.metadata.create_all(engine)
    with Session(engine) as db:
        assert db.scalar(select(Run)).treasures == 42
        assert db.scalar(select(func.count()).select_from(RunSubmission)) == 0
    engine.dispose()


@pytest.mark.skipif(not os.getenv("TEST_POSTGRES_URL"), reason="CI PostgreSQL service required")
@pytest.mark.parametrize("conflicting", [False, True])
def test_concurrent_submissions_are_atomic(conflicting):
    engine = create_engine(os.environ["TEST_POSTGRES_URL"])
    Base.metadata.create_all(engine)
    key, name = uuid4(), uuid4().hex
    barrier = Barrier(2)

    @event.listens_for(engine, "before_cursor_execute")
    def synchronize_inserts(conn, cursor, statement, parameters, context, executemany):
        # Both requests passed the initial lookup and inserted their Run.
        # Force the unique-key race instead of relying on thread scheduling.
        if statement.startswith("INSERT INTO run_submissions"):
            barrier.wait(timeout=10)

    def send(index):
        with Session(engine, expire_on_commit=False) as db:
            response = Response(status_code=201)
            payload = RunSubmit(submission_id=key, player_name=name,
                                treasures=100 + (index if conflicting else 0), level=3)
            try:
                result = submit_run(payload, response, db)
                return response.status_code, result.id
            except HTTPException as exc:
                return exc.status_code, None

    try:
        with ThreadPoolExecutor(max_workers=2) as pool:
            results = list(pool.map(send, range(2)))
        assert sorted(status for status, _ in results) == ([201, 409] if conflicting else [200, 201])
        if not conflicting:
            assert results[0][1] == results[1][1]
        with Session(engine) as db:
            assert db.scalar(select(func.count()).select_from(Run).where(Run.player_name == name)) == 1
            assert db.get(RunSubmission, str(key)) is not None
    finally:
        engine.dispose()
