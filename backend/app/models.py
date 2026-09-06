from datetime import UTC, datetime

from sqlalchemy import DateTime, ForeignKey, Integer, String
from sqlalchemy.orm import Mapped, mapped_column

from app.database import Base


class Run(Base):
    """Результат одного забега, присланный игрой после смерти или победы."""

    __tablename__ = "runs"

    id: Mapped[int] = mapped_column(Integer, primary_key=True)
    player_name: Mapped[str] = mapped_column(String(32), default="anonymous")
    treasures: Mapped[int] = mapped_column(Integer)
    level: Mapped[int] = mapped_column(Integer)
    enemies_killed: Mapped[int] = mapped_column(Integer, default=0)
    food_used: Mapped[int] = mapped_column(Integer, default=0)
    elixirs_used: Mapped[int] = mapped_column(Integer, default=0)
    scrolls_read: Mapped[int] = mapped_column(Integer, default=0)
    attacks_made: Mapped[int] = mapped_column(Integer, default=0)
    hits_taken: Mapped[int] = mapped_column(Integer, default=0)
    tiles_moved: Mapped[int] = mapped_column(Integer, default=0)
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), default=lambda: datetime.now(UTC)
    )


class RunSubmission(Base):
    """Idempotency keys in a new table: existing runs need no schema changes."""

    __tablename__ = "run_submissions"

    id: Mapped[str] = mapped_column(String(36), primary_key=True)
    run_id: Mapped[int] = mapped_column(ForeignKey("runs.id"), unique=True)


class RankedTicket(Base):
    """Server-issued capability for one run, never exposed in the leaderboard."""
    __tablename__ = "ranked_tickets"

    id: Mapped[str] = mapped_column(String(36), primary_key=True)
    seed: Mapped[str] = mapped_column(String(20))
    player_name: Mapped[str] = mapped_column(String(32))
    version: Mapped[str] = mapped_column(String(64))
    expires_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), index=True)


class RankedResult(Base):
    __tablename__ = "ranked_results"

    ticket_id: Mapped[str] = mapped_column(ForeignKey("ranked_tickets.id"), primary_key=True)
    run_id: Mapped[int] = mapped_column(ForeignKey("runs.id"), unique=True)
    actions_hash: Mapped[str] = mapped_column(String(64))
