from uuid import uuid4

import pytest
from pydantic import ValidationError

from app.schemas import RunSubmit


def test_submission_allows_ability_replay_alphabet():
    actions = "btwtdtat sj0k0".replace(" ", "")
    assert RunSubmit(ticket=uuid4(), actions=actions).actions == actions


@pytest.mark.parametrize("actions", ["<script>", "t🗡", "b;", "t\n"])
def test_submission_rejects_non_action_characters(actions):
    with pytest.raises(ValidationError):
        RunSubmit(ticket=uuid4(), actions=actions)
