from unittest.mock import AsyncMock

import pytest

from backend.application.exceptions import NotFoundError, PermissionDeniedError
from backend.application.ports.call_session_store import CallSessionStore
from backend.application.use_cases.calls import CallUseCase
from backend.domain.call import CallSession, CallStatus, CallType


class InMemoryCallStore(CallSessionStore):
    def __init__(self) -> None:
        self.sessions: dict[str, CallSession] = {}

    async def save(self, session: CallSession) -> None:
        self.sessions[session.call_id] = session

    async def get(self, call_id: str) -> CallSession | None:
        return self.sessions.get(call_id)

    async def delete(self, call_id: str) -> None:
        self.sessions.pop(call_id, None)


@pytest.mark.application
@pytest.mark.asyncio
async def test_start_call_checks_interaction_policy_and_returns_dto() -> None:
    messaging = AsyncMock()
    messaging.can_send_message.return_value = True
    store = InMemoryCallStore()

    result = await CallUseCase(store, messaging).start(1, 2, CallType.VIDEO)

    assert result.caller_id == 1
    assert result.callee_id == 2
    assert result.call_type == "video"
    assert result.status == CallStatus.RINGING
    messaging.can_send_message.assert_awaited_once_with(1, 2)


@pytest.mark.application
@pytest.mark.asyncio
async def test_start_call_rejects_forbidden_target_without_persisting() -> None:
    messaging = AsyncMock()
    messaging.can_send_message.return_value = False
    store = InMemoryCallStore()

    with pytest.raises(PermissionDeniedError):
        await CallUseCase(store, messaging).start(1, 2, CallType.AUDIO)

    assert store.sessions == {}


@pytest.mark.application
@pytest.mark.asyncio
async def test_call_use_case_authorizes_signaling_and_lifecycle() -> None:
    messaging = AsyncMock()
    messaging.can_send_message.return_value = True
    store = InMemoryCallStore()
    use_case = CallUseCase(store, messaging)
    started = await use_case.start(1, 2, CallType.AUDIO)

    with pytest.raises(PermissionDeniedError):
        await use_case.authorize_signaling(started.call_id, 3)

    accepted = await use_case.accept(started.call_id, 2)
    assert accepted.status == CallStatus.ACTIVE
    ended = await use_case.end(started.call_id, 1)
    assert ended.status == CallStatus.ENDED


@pytest.mark.application
@pytest.mark.asyncio
async def test_missing_call_is_not_found() -> None:
    use_case = CallUseCase(InMemoryCallStore(), AsyncMock())

    with pytest.raises(NotFoundError):
        await use_case.authorize_signaling("missing", 1)
