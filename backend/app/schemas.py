from datetime import datetime
import unicodedata
from typing import Annotated
from uuid import UUID

from pydantic import BaseModel, Field, field_validator


Counter = Annotated[int, Field(ge=0, le=2**31 - 1)]


class RunFields(BaseModel):
    """Поля счёта в пределах типа INTEGER в PostgreSQL."""

    player_name: str = Field(default="anonymous", min_length=1, max_length=32)
    treasures: Counter
    level: int = Field(ge=1, le=21)
    enemies_killed: Counter = 0
    food_used: Counter = 0
    elixirs_used: Counter = 0
    scrolls_read: Counter = 0
    attacks_made: Counter = 0
    hits_taken: Counter = 0
    tiles_moved: Counter = 0


# Столько символов, сколько игра даёт набрать и показывает.
PLAYER_NAME_LENGTH = 16


def clean_player_name(name: str) -> str:
    """Убирает управляющие и невидимые символы (нулевой ширины, смены направления),
    обрезает пробелы по краям и оставляет не больше PLAYER_NAME_LENGTH символов;
    если ничего не осталось, имя "anonymous", как в игре."""
    kept = "".join(ch for ch in name if not unicodedata.category(ch).startswith("C"))
    return kept.strip()[:PLAYER_NAME_LENGTH].strip() or "anonymous"


class RunStart(BaseModel):
    player_name: str = Field(default="anonymous", min_length=1, max_length=32)
    version: str = Field(min_length=1, max_length=64)
    model_config = {"extra": "forbid"}

    @field_validator("player_name")
    @classmethod
    def clean_name(cls, name: str) -> str:
        return clean_player_name(name)


class RunTicket(BaseModel):
    ticket: UUID
    seed: str
    version: str


class RunSubmit(BaseModel):
    ticket: UUID
    actions: str = Field(min_length=1, max_length=60000, pattern=r"^[wasdWASDzhjkebt0-9]+$")
    model_config = {"extra": "forbid"}


class RunOut(RunFields):
    """Запись лидерборда в ответах API."""

    id: int
    created_at: datetime
    # Старые записи владелец проекта признал доверенными.
    # Новые записи по-прежнему требуют повтора на сервере; это не разрешение на запись.
    verified: bool = True

    model_config = {"from_attributes": True}


class RunSubmitted(RunOut):
    """Отправленный забег с его местом в таблице рекордов."""

    place: int = Field(ge=1)
    # Золото 10-го места или None, пока забегов меньше десяти.
    top10_gold: int | None = None
