from datetime import UTC, datetime, timedelta
from hashlib import sha256
import secrets
from uuid import uuid4

from fastapi import APIRouter, Depends, HTTPException, Query, Response
from sqlalchemy import desc, select
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session

from app.database import get_db
from app.models import RankedResult, RankedTicket, Run
from app.schemas import RunOut, RunStart, RunSubmit, RunTicket
from app.verifier import rules_version, verify

router = APIRouter(prefix="/api", tags=["leaderboard"])


@router.post("/runs/start", response_model=RunTicket, status_code=201)
def start_run(payload: RunStart, db: Session = Depends(get_db)):
    version = rules_version()
    if payload.version != version:
        raise HTTPException(409, "Game rules changed; reload the game")
    ticket = RankedTicket(id=str(uuid4()), seed=str(secrets.randbelow(2**63 - 1) + 1),
                          player_name=payload.player_name, version=version,
                          expires_at=datetime.now(UTC) + timedelta(hours=24))
    db.add(ticket)
    db.commit()
    return RunTicket(ticket=ticket.id, seed=ticket.seed, version=version)


def result_out(run: Run, verified: bool) -> RunOut:
    return RunOut.model_validate(run).model_copy(update={"verified": verified})


@router.post("/runs", response_model=RunOut, status_code=201)
def submit_run(payload: RunSubmit, response: Response, db: Session = Depends(get_db)):
    """Only server-recomputed terminal runs may create leaderboard records."""
    key = str(payload.ticket)
    digest = sha256(payload.actions.encode("ascii")).hexdigest()

    def replay():
        saved = db.get(RankedResult, key)
        if saved is None:
            return None
        if saved.actions_hash != digest:
            raise HTTPException(409, "Ticket already used for another replay")
        response.status_code = 200
        return result_out(db.get(Run, saved.run_id), True)

    existing = replay()
    if existing is not None:
        return existing
    ticket = db.get(RankedTicket, key)
    if ticket is None:
        raise HTTPException(404, "Unknown run ticket")
    # SQLite returns naive datetimes; PostgreSQL preserves the UTC timezone.
    expires = ticket.expires_at.replace(tzinfo=UTC)
    if expires <= datetime.now(UTC):
        raise HTTPException(410, "Run ticket expired")
    if ticket.version != rules_version():
        raise HTTPException(409, "Game rules changed; this run cannot be verified")

    values = verify(ticket.seed, payload.actions)
    run = Run(player_name=ticket.player_name, **values)
    db.add(run)
    try:
        db.flush()
        db.add(RankedResult(ticket_id=key, run_id=run.id, actions_hash=digest))
        db.commit()
    except IntegrityError:
        # A racing request may have claimed the ticket. Roll back its extra
        # score as well, and return only the already committed result.
        db.rollback()
        existing = replay()
        if existing is not None:
            return existing
        raise
    db.refresh(run)
    return result_out(run, True)


@router.get("/leaderboard", response_model=list[RunOut])
def get_leaderboard(limit: int = Query(default=10, ge=1, le=100), db: Session = Depends(get_db)):
    """Keep legacy scores visible, explicitly distinguish verified new runs."""
    stmt = (select(Run, RankedResult.run_id)
            .outerjoin(RankedResult, RankedResult.run_id == Run.id)
            .order_by(desc(Run.treasures), desc(Run.level)).limit(limit))
    return [result_out(run, verified_id is not None) for run, verified_id in db.execute(stmt)]
