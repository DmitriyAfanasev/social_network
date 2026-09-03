from typing import Any
from unittest.mock import AsyncMock

import pytest

from backend.application.exceptions import PermissionDeniedError
from backend.application.use_cases.messages import MessagingUseCase
from backend.domain.user.entity import User
from backend.presentation.messages.ws.constants import (
    TYPING_EVENT,
    WEBSOCKET_PING_EVENT,
    WEBSOCKET_PONG_EVENT,
)
from backend.presentation.messages.ws.handlers import (
    handle_socket_event,
    subscribe_to_conversation,
)
from backend.presentation.messages.ws.ports import WebSocketPayload, WebSocketSender
from tests.fakes import FakeTransaction


class FakeWebSocket(WebSocketSender):
    def __init__(self) -> None:
        self._cookies: dict[str, str] = {}
        self.sent: list[WebSocketPayload] = []

    @property
    def cookies(self) -> dict[str, str]:
        return self._cookies

    async def send_json(self, payload: Any) -> None:
        assert isinstance(payload, dict)
        self.sent.append(payload)


def make_user(user_id: int) -> User:
    return User(
        id=user_id,
        username=f"user{user_id}",
        email=f"user{user_id}@example.com",
        hashed_password="hash",
    )


def make_messaging_use_case(*, blocked: bool = False) -> MessagingUseCase:
    repository = AsyncMock()
    repository.get_participant_ids.return_value = [1, 2]
    profiles = AsyncMock()
    profiles.get_by_id.return_value = make_user(2)
    friends = AsyncMock()
    friends.is_friend.return_value = False
    friends.are_friends_of_friends.return_value = False
    blocks = AsyncMock()
    blocks.is_blocked.return_value = blocked
    return MessagingUseCase(
        repository,
        AsyncMock(),
        FakeTransaction(),
        profile_repository=profiles,
        friend_repository=friends,
        block_repository=blocks,
    )


@pytest.mark.application
@pytest.mark.asyncio
async def test_websocket_subscription_checks_interaction_policy() -> None:
    websocket = FakeWebSocket()
    manager = AsyncMock()
    use_case = make_messaging_use_case(blocked=True)

    with pytest.raises(PermissionDeniedError):
        await subscribe_to_conversation(
            websocket,
            10,
            1,
            use_case,
            manager,
            set(),
        )

    manager.subscribe.assert_not_awaited()


@pytest.mark.application
@pytest.mark.asyncio
async def test_websocket_typing_checks_policy_before_broadcast() -> None:
    websocket = FakeWebSocket()
    manager = AsyncMock()
    use_case = make_messaging_use_case(blocked=True)

    event = {"type": TYPING_EVENT, "conversation_id": 10, "is_typing": True}
    handler = handle_socket_event
    with pytest.raises(PermissionDeniedError):
        await handler(
            websocket,
            event,
            1,
            use_case,
            manager,
            AsyncMock(),
            {10},
        )

    manager.broadcast.assert_not_awaited()


@pytest.mark.application
@pytest.mark.asyncio
async def test_websocket_ping_updates_activity_through_use_case() -> None:
    websocket = FakeWebSocket()
    manager = AsyncMock()
    use_case = AsyncMock()
    activity = AsyncMock()

    await handle_socket_event(
        websocket,
        {"type": WEBSOCKET_PING_EVENT},
        1,
        use_case,
        manager,
        activity,
        set(),
    )

    activity.execute.assert_awaited_once_with(1)
    assert websocket.sent == [{"type": WEBSOCKET_PONG_EVENT}]
