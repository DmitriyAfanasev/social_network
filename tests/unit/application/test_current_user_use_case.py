from unittest.mock import AsyncMock, Mock

import pytest

from backend.application.exceptions import AuthenticationError
from backend.application.use_cases.auth import GetCurrentUserUseCase
from backend.domain.user.entity import User


# TODO: если make_user/target_user начнут повторяться в нескольких файлах,
def make_user(*, active: bool = True) -> User:
    return User(
        id=4,
        username="alice",
        email="alice@example.com",
        hashed_password="hash",
        is_active=active,
    )


@pytest.mark.application
@pytest.mark.asyncio
async def test_current_user_resolves_token_and_updates_activity() -> None:
    repository = Mock()
    repository.get_by_id = AsyncMock(return_value=make_user())
    repository.touch_last_seen = AsyncMock()
    token_service = Mock()
    token_service.get_access_user_id.return_value = 4

    result = await GetCurrentUserUseCase(repository, token_service).execute("token")

    assert result is not None
    assert result.id == 4
    token_service.get_access_user_id.assert_called_once_with("token")
    repository.touch_last_seen.assert_awaited_once_with(4)


@pytest.mark.application
@pytest.mark.asyncio
@pytest.mark.parametrize("token", [None, ""])
async def test_required_current_user_rejects_missing_token(token: str | None) -> None:
    with pytest.raises(AuthenticationError):
        await GetCurrentUserUseCase(Mock(), Mock()).execute(token)


@pytest.mark.application
@pytest.mark.asyncio
async def test_optional_current_user_ignores_invalid_token() -> None:
    token_service = Mock()
    token_service.get_access_user_id.side_effect = ValueError("invalid")

    result = await GetCurrentUserUseCase(Mock(), token_service).execute(
        "token", required=False
    )

    assert result is None


@pytest.mark.application
@pytest.mark.asyncio
async def test_current_user_rejects_inactive_user_without_touching_activity() -> None:
    repository = Mock()
    repository.get_by_id = AsyncMock(return_value=make_user(active=False))
    repository.touch_last_seen = AsyncMock()
    token_service = Mock()
    token_service.get_access_user_id.return_value = 4

    with pytest.raises(AuthenticationError):
        await GetCurrentUserUseCase(repository, token_service).execute("token")

    repository.touch_last_seen.assert_not_awaited()
