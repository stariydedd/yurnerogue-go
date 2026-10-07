from datetime import UTC, datetime, timedelta
from hashlib import sha256
import secrets
from uuid import uuid4

from fastapi import APIRouter, Depends, HTTPException, Query, Response
from sqlalchemy import and_, delete, desc, exists, func, or_, select
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session

from app.database import get_db
from app.models import RankedResult, RankedTicket, Run
from app.schemas import RunOut, RunStart, RunSubmit, RunSubmitted, RunTicket
from app.verifier import rules_version, verify

router = APIRouter(prefix="/api", tags=["leaderboard"])

TICKET_LIFETIME = timedelta(hours=24)
# Хранится какое-то время после истечения, чтобы поздняя отправка всё ещё получала «истёк».
EXPIRED_TICKET_GRACE = timedelta(hours=1)


def drop_expired_tickets(db: Session, now: datetime) -> None:
    """Каждый старт добавляет билет, а брошенные забеги свои не используют. Билеты
    отправленных забегов остаются: в них имя, под которым забег сыгран."""
    db.execute(delete(RankedTicket).where(
        RankedTicket.expires_at < now - EXPIRED_TICKET_GRACE,
        ~exists().where(RankedResult.ticket_id == RankedTicket.id),
    ))


@router.post("/runs/start", response_model=RunTicket, status_code=201)
def start_run(payload: RunStart, db: Session = Depends(get_db)):
    version = rules_version()
    if payload.version != version:
        raise HTTPException(409, "Game rules changed; reload the game")
    now = datetime.now(UTC)
    drop_expired_tickets(db, now)
    ticket = RankedTicket(id=str(uuid4()), seed=str(secrets.randbelow(2**63 - 1) + 1),
                          player_name=payload.player_name, version=version,
                          expires_at=now + TICKET_LIFETIME)
    db.add(ticket)
    db.commit()
    return RunTicket(ticket=ticket.id, seed=ticket.seed, version=version)


def result_out(run: Run) -> RunOut:
    return RunOut.model_validate(run)


# Единый порядок таблицы рекордов: золото, затем глубина, затем кто закончил раньше.
LEADERBOARD_ORDER = (desc(Run.treasures), desc(Run.level), Run.id)


def submitted_out(db: Session, run: Run) -> RunSubmitted:
    """Забег с его местом в LEADERBOARD_ORDER и золотом 10-го места."""
    ahead = db.scalar(select(func.count()).select_from(Run).where(or_(
        Run.treasures > run.treasures,
        and_(Run.treasures == run.treasures, Run.level > run.level),
        and_(Run.treasures == run.treasures, Run.level == run.level, Run.id < run.id),
    )))
    tenth = db.scalar(select(Run.treasures).order_by(*LEADERBOARD_ORDER).offset(9).limit(1))
    return RunSubmitted(**RunOut.model_validate(run).model_dump(), place=ahead + 1, top10_gold=tenth)


@router.post("/runs", response_model=RunSubmitted, status_code=201)
def submit_run(payload: RunSubmit, response: Response, db: Session = Depends(get_db)):
    """Записи в таблице рекордов создают только завершённые забеги, пересчитанные сервером."""
    key = str(payload.ticket)
    digest = sha256(payload.actions.encode("ascii")).hexdigest()

    def replay():
        saved = db.get(RankedResult, key)
        if saved is None:
            return None
        if saved.actions_hash != digest:
            raise HTTPException(409, "Ticket already used for another replay")
        response.status_code = 200
        return submitted_out(db, db.get(Run, saved.run_id))

    existing = replay()
    if existing is not None:
        return existing
    ticket = db.get(RankedTicket, key)
    if ticket is None:
        raise HTTPException(404, "Unknown run ticket")
    # SQLite возвращает даты без часового пояса; PostgreSQL сохраняет UTC.
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
        # Параллельный запрос мог уже занять билет. Откатываем и его лишний
        # результат и возвращаем только уже сохранённый.
        db.rollback()
        existing = replay()
        if existing is not None:
            return existing
        raise
    db.refresh(run)
    return submitted_out(db, run)


@router.get("/leaderboard", response_model=list[RunOut])
def get_leaderboard(limit: int = Query(default=10, ge=1, le=100), db: Session = Depends(get_db)):
    """Старые результаты доверенные по решению владельца; новые записи требуют повтора."""
    stmt = select(Run).order_by(*LEADERBOARD_ORDER).limit(limit)
    return [result_out(run) for run in db.scalars(stmt)]
