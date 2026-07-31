from datetime import UTC, datetime

from sqlalchemy import DateTime, Integer, String
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
