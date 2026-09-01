from datetime import UTC, datetime
from types import SimpleNamespace
from unittest.mock import AsyncMock

import pytest

from backend.application.exceptions import NotFoundError, PermissionDeniedError, ValidationAppError
from backend.application.use_cases.messages import MessagingUseCase, decode_cursor, encode_cursor
from tests.fakes import FakeTransaction


def make_use_case() -> tuple[MessagingUseCase, AsyncMock, AsyncMock]:
    repository = AsyncMock()
    outbox = AsyncMock()
    return MessagingUseCase(repository, outbox, FakeTransaction()), repository, outbox


@pytest.mark.application
def test_cursor_round_trip_preserves_datetime_and_id() -> None:
    created_at = datetime(2026, 1, 2, 3, 4, tzinfo=UTC)

    assert decode_cursor(encode_cursor(created_at, 7)) == (created_at, 7)


@pytest.mark.application
@pytest.mark.parametrize("cursor", ["bad", "e30", "!!!"])
def test_invalid_cursor_is_application_validation_error(cursor: str) -> None:
    with pytest.raises(ValidationAppError, match="курсор"):
        decode_cursor(cursor)


@pytest.mark.application
@pytest.mark.asyncio
async def test_list_messages_validates_paging_and_builds_next_cursor() -> None:
    use_case, repository, _ = make_use_case()
    message = SimpleNamespace(id=3, created_at=datetime.now(UTC))
    repository.list_messages.return_value = ([message], True)

    result = await use_case.list_messages(10, 1, 0, 20)

    assert result.has_more is True
    assert decode_cursor(result.next_cursor) == (message.created_at, 3)
    with pytest.raises(ValidationAppError):
        await use_case.list_messages(10, 1, -1, 20)
    with pytest.raises(ValidationAppError):
        await use_case.list_messages(10, 1, 0, 101)


@pytest.mark.application
@pytest.mark.asyncio
async def test_send_message_rejects_empty_or_too_long_text_without_writing() -> None:
    use_case, repository, _ = make_use_case()
    with pytest.raises(ValidationAppError):
        await use_case.send_message(10, 1, "   ")
    with pytest.raises(ValidationAppError):
        await use_case.send_message(10, 1, "x" * 5001)
    repository.create_message.assert_not_awaited()


@pytest.mark.application
@pytest.mark.asyncio
async def test_send_message_normalizes_text_and_records_outbox_event() -> None:
    use_case, repository, outbox = make_use_case()
    repository.get_participant_ids.return_value = [1]
    message = SimpleNamespace(id=8)
    repository.create_message.return_value = message

    result = await use_case.send_message(10, 1, "  hello  ")

    assert result.message is message
    repository.create_message.assert_awaited_once_with(10, 1, "hello", None)
    outbox.add.assert_awaited_once()


@pytest.mark.application
@pytest.mark.asyncio
async def test_direct_conversation_rejects_self_and_checks_policy() -> None:
    use_case, repository, _ = make_use_case()
    with pytest.raises(ValidationAppError):
        await use_case.get_or_create_direct(1, 1)

    use_case.profile_repository = AsyncMock()
    use_case.friend_repository = AsyncMock()
    use_case.block_repository = AsyncMock()
    use_case.block_repository.is_blocked.return_value = True
    use_case.profile_repository.get_by_id.return_value = SimpleNamespace(profile=None)
    with pytest.raises(PermissionDeniedError):
        await use_case.get_or_create_direct(1, 2)
    repository.get_or_create_direct.assert_not_awaited()


@pytest.mark.application
@pytest.mark.asyncio
async def test_message_mutations_are_transactional_and_mark_read_reports_missing() -> None:
    use_case, repository, outbox = make_use_case()
    repository.update_message.return_value = SimpleNamespace(id=2)
    result = await use_case.edit_message(10, 2, 1, " changed ")
    assert result.message.id == 2
    repository.update_message.assert_awaited_once_with(10, 2, 1, "changed")
    outbox.add.assert_awaited_once()

    repository.mark_read.return_value = False
    with pytest.raises(NotFoundError):
        await use_case.mark_read(10, 1, 2)
