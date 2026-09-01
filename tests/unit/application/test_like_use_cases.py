from unittest.mock import AsyncMock

import pytest

from backend.application.results import ToggleLikeResult
from backend.application.use_cases.comment_likes import ToggleCommentLikeUseCase
from backend.application.use_cases.likes import TogglePostLikeUseCase
from backend.domain.user.entity import User
from tests.fakes import FakeTransaction


def make_user() -> User:
    return User(id=1, username="user", email="user@example.com", hashed_password="hash")


@pytest.mark.application
@pytest.mark.asyncio
@pytest.mark.parametrize("use_case_type", [TogglePostLikeUseCase, ToggleCommentLikeUseCase])
async def test_toggle_like_returns_repository_state_and_emits_event(use_case_type: type) -> None:
    repository = AsyncMock()
    outbox = AsyncMock()
    repository.toggle_post_like.return_value = ToggleLikeResult(True, "liked", 4, True)
    repository.toggle_comment_like.return_value = ToggleLikeResult(True, "liked", 4, True)
    use_case = use_case_type(repository, outbox, FakeTransaction())

    result = await use_case.execute(make_user(), 9)

    assert result == ToggleLikeResult(True, "liked", 4, True)
    outbox.add.assert_awaited_once()
