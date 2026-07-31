from datetime import datetime

from pydantic import BaseModel, Field


class RunSubmit(BaseModel):
    """Тело POST /api/runs — результат забега от игры."""

    player_name: str = Field(default="anonymous", min_length=1, max_length=32)
    treasures: int = Field(ge=0)
    level: int = Field(ge=1, le=21)
    enemies_killed: int = Field(default=0, ge=0)
    food_used: int = Field(default=0, ge=0)
    elixirs_used: int = Field(default=0, ge=0)
    scrolls_read: int = Field(default=0, ge=0)
    attacks_made: int = Field(default=0, ge=0)
    hits_taken: int = Field(default=0, ge=0)
    tiles_moved: int = Field(default=0, ge=0)


class RunOut(RunSubmit):
    """Запись лидерборда в ответах API."""

    id: int
    created_at: datetime

    model_config = {"from_attributes": True}
