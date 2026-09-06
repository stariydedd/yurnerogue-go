from fastapi import APIRouter, Depends, HTTPException, Query, Response
from sqlalchemy import desc, select
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session

from app.database import get_db
from app.models import Run, RunSubmission
from app.schemas import RunOut, RunSubmit

router = APIRouter(prefix="/api", tags=["leaderboard"])


@router.post("/runs", response_model=RunOut, status_code=201)
def submit_run(payload: RunSubmit, response: Response, db: Session = Depends(get_db)):
    """Save once per submission ID, including concurrent retries."""
    values = payload.model_dump(exclude={"submission_id"})
    key = str(payload.submission_id) if payload.submission_id else None

    def replay():
        submission = db.get(RunSubmission, key)
        if submission is None:
            return None
        existing = db.get(Run, submission.run_id)
        if any(getattr(existing, field) != value for field, value in values.items()):
            raise HTTPException(409, "submission_id already used for another result")
        response.status_code = 200
        return existing

    if key:
        existing = replay()
        if existing is not None:
            return existing

    run = Run(**values)
    db.add(run)
    try:
        if key:
            db.flush()
            db.add(RunSubmission(id=key, run_id=run.id))
        db.commit()
    except IntegrityError:
        # Roll back the extra run too if another request won the unique key.
        db.rollback()
        if key:
            existing = replay()
            if existing is not None:
                return existing
        raise
    db.refresh(run)
    return run


@router.get("/leaderboard", response_model=list[RunOut])
def get_leaderboard(limit: int = Query(default=10, ge=1, le=100), db: Session = Depends(get_db)):
    """Топ забегов: по золоту, затем по достигнутому уровню."""
    stmt = select(Run).order_by(desc(Run.treasures), desc(Run.level)).limit(limit)
    return db.scalars(stmt).all()
