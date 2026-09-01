from unittest.mock import AsyncMock

import pytest

from backend.application.event_types import FRIEND_REQUESTED_EVENT
from backend.application.use_cases.friends import AddFriendUseCase
from backend.domain.user.entity import User


class FakeTransaction:
    async def __aenter__(self) -> "FakeTransaction":
        return self

    async def __aexit__(self, exc_type: object, exc: object, traceback: object) -> None:
        return None


@pytest.mark.asyncio
async def test_friend_request_is_written_to_outbox_in_same_use_case() -> None:
    repository = AsyncMock()
    repository.user_exists.return_value = True
    repository.is_subscribed.return_value = False
    outbox = AsyncMock()
    current_user = User(
        id=1,
        username="sender",
        email="sender@example.com",
        hashed_password="hash",
    )

    result = await AddFriendUseCase(
        repository,
        outbox,
        FakeTransaction(),
    ).execute(current_user, 2)

    assert result.is_friend is False
    event = outbox.add.await_args.args[0]
    assert event.event_type == FRIEND_REQUESTED_EVENT
    assert event.payload["user_id"] == 1
    assert event.payload["entity_id"] == 2
