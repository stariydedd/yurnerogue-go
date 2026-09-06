from datetime import datetime
from typing import Annotated
from uuid import UUID

from pydantic import BaseModel, Field


Counter = Annotated[int, Field(ge=0, le=2**31 - 1)]


class RunFields(BaseModel):
    """Score fields bounded by the PostgreSQL INTEGER storage type."""

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


class RunSubmit(RunFields):
    # Optional for older clients; new clients reuse one UUID for the whole run.
    submission_id: UUID | None = None


class RunOut(RunFields):
    """Запись лидерборда в ответах API."""

    id: int
    created_at: datetime

    model_config = {"from_attributes": True}
