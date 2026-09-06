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


class RunStart(BaseModel):
    player_name: str = Field(default="anonymous", min_length=1, max_length=32)
    version: str = Field(min_length=1, max_length=64)
    model_config = {"extra": "forbid"}


class RunTicket(BaseModel):
    ticket: UUID
    seed: str
    version: str


class RunSubmit(BaseModel):
    ticket: UUID
    actions: str = Field(min_length=1, max_length=60000, pattern=r"^[wasdWASDzhjke0-9]+$")
    model_config = {"extra": "forbid"}


class RunOut(RunFields):
    """Запись лидерборда в ответах API."""

    id: int
    created_at: datetime
    # Legacy records are grandfathered as trusted by the project owner.
    # New records still require server replay; this is not a write permission.
    verified: bool = True

    model_config = {"from_attributes": True}
