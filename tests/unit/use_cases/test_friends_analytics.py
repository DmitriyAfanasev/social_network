from unittest.mock import AsyncMock

import pytest

from backend.application.event_types import FRIEND_REQUESTED_EVENT
from backend.application.use_cases.friends import AddFriendUseCase
from backend.domain.user.entity import User
from tests.fakes import FakeTransaction


@pytest.mark.asyncio
async def test_friend_request_is_written_to_outbox_in_same_use_case() -> None:
    repository = AsyncMock()
    repository.user_exists.return_value = True
    repository.is_subscribed.return_value = False
    outbox = AsyncMock()
    block_repository = AsyncMock()
    block_repository.is_blocked.return_value = False
    profile_repository = AsyncMock()
    profile_repository.get_by_id.return_value = User(
        id=2,
        username="target",
        email="target@example.com",
        hashed_password="hash",
    )
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
        block_repository,
        profile_repository,
    ).execute(current_user, 2)

    assert result.is_friend is False
    event = outbox.add.await_args.args[0]
    assert event.event_type == FRIEND_REQUESTED_EVENT
    assert event.payload["user_id"] == 1
    assert event.payload["entity_id"] == 2
