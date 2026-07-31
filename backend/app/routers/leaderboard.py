from fastapi import APIRouter, Depends, Query
from sqlalchemy import desc, select
from sqlalchemy.orm import Session

from app.database import get_db
from app.models import Run
from app.schemas import RunOut, RunSubmit

router = APIRouter(prefix="/api", tags=["leaderboard"])


@router.post("/runs", response_model=RunOut, status_code=201)
def submit_run(payload: RunSubmit, db: Session = Depends(get_db)):
    """Сохраняет результат забега."""
    run = Run(**payload.model_dump())
    db.add(run)
    db.commit()
    db.refresh(run)
    return run


@router.get("/leaderboard", response_model=list[RunOut])
def get_leaderboard(limit: int = Query(default=10, ge=1, le=100), db: Session = Depends(get_db)):
    """Топ забегов: по золоту, затем по достигнутому уровню."""
    stmt = select(Run).order_by(desc(Run.treasures), desc(Run.level)).limit(limit)
    return db.scalars(stmt).all()
