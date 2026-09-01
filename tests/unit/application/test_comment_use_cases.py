from datetime import UTC, datetime
from types import SimpleNamespace
from unittest.mock import AsyncMock

import pytest

from backend.application.commands import CreateCommentCommand, UpdateCommentCommand
from backend.application.exceptions import NotFoundError, PermissionDeniedError, ValidationAppError
from backend.application.use_cases.comments import (
    CreateCommentUseCase,
    DeleteCommentUseCase,
    GetCommentsUseCase,
    UpdateCommentUseCase,
)
from backend.domain.user.entity import User
from tests.fakes import FakeTransaction


def make_user(user_id: int = 1) -> User:
    return User(id=user_id, username=f"user{user_id}", email=f"u{user_id}@e.com", hashed_password="hash")


def make_post(author_id: int = 1) -> SimpleNamespace:
    return SimpleNamespace(author_id=author_id)


def make_comment(*, user_id: int = 1, post_id: int = 10, parent_id: int | None = None) -> SimpleNamespace:
    now = datetime.now(UTC)
    return SimpleNamespace(id=5, user_id=user_id, post_id=post_id, parent_id=parent_id, text="old", created_at=now, updated_at=now)


@pytest.mark.application
@pytest.mark.asyncio
async def test_create_comment_rejects_missing_post_or_blocked_author() -> None:
    repository = AsyncMock()
    blocks = AsyncMock()
    use_case = CreateCommentUseCase(repository, AsyncMock(), FakeTransaction(), blocks)
    repository.get_post_by_id.return_value = None
    with pytest.raises(NotFoundError):
        await use_case.execute(make_user(), 10, CreateCommentCommand("hello"))

    repository.get_post_by_id.return_value = make_post(author_id=2)
    blocks.is_blocked.return_value = True
    with pytest.raises(PermissionDeniedError):
        await use_case.execute(make_user(), 10, CreateCommentCommand("hello"))
    repository.create.assert_not_awaited()


@pytest.mark.application
@pytest.mark.asyncio
async def test_create_comment_validates_parent_post_and_depth() -> None:
    repository = AsyncMock()
    repository.get_post_by_id.return_value = make_post()
    repository.get_comment_by_id.return_value = make_comment(post_id=99)
    repository.get_comment_depth.return_value = 0
    blocks = AsyncMock()
    blocks.is_blocked.return_value = False
    use_case = CreateCommentUseCase(repository, AsyncMock(), FakeTransaction(), blocks)

    with pytest.raises(ValidationAppError):
        await use_case.execute(make_user(), 10, CreateCommentCommand("reply", parent_id=5))


@pytest.mark.application
@pytest.mark.asyncio
async def test_get_comments_advances_offset_by_limit() -> None:
    repository = AsyncMock()
    repository.get_paginated.return_value = (["comment"], True)

    result = await GetCommentsUseCase(repository).execute(10, offset=5, limit=2)

    assert result.offset == 7
    assert result.has_more is True
    repository.get_paginated.assert_awaited_once_with(10, 5, 2)


@pytest.mark.application
@pytest.mark.asyncio
async def test_update_comment_requires_owner_and_nonblank_text() -> None:
    repository = AsyncMock()
    repository.get_comment_by_id.return_value = make_comment(user_id=1)
    repository.update.return_value = make_comment(user_id=1)
    use_case = UpdateCommentUseCase(repository, FakeTransaction())

    with pytest.raises(ValidationAppError):
        await use_case.execute(make_user(), 5, UpdateCommentCommand("  "))
    with pytest.raises(PermissionDeniedError):
        await use_case.execute(make_user(2), 5, UpdateCommentCommand("new"))
    await use_case.execute(make_user(), 5, UpdateCommentCommand(" new "))
    repository.update.assert_awaited_once_with(repository.get_comment_by_id.return_value, "new")


@pytest.mark.application
@pytest.mark.asyncio
async def test_delete_comment_allows_owner_post_owner_and_moderator() -> None:
    repository = AsyncMock()
    repository.get_comment_by_id.return_value = make_comment(user_id=2)
    repository.get_post_by_id.return_value = make_post(author_id=3)
    authorization = AsyncMock()
    authorization.has_permission.return_value = True
    audit = AsyncMock()

    await DeleteCommentUseCase(repository, FakeTransaction(), authorization, audit).execute(make_user(1), 5)

    repository.delete.assert_awaited_once()
    audit.record.assert_awaited_once()


@pytest.mark.application
@pytest.mark.asyncio
async def test_delete_comment_rejects_unprivileged_user() -> None:
    repository = AsyncMock()
    repository.get_comment_by_id.return_value = make_comment(user_id=2)
    repository.get_post_by_id.return_value = make_post(author_id=3)
    authorization = AsyncMock()
    authorization.has_permission.return_value = False

    with pytest.raises(PermissionDeniedError):
        await DeleteCommentUseCase(repository, FakeTransaction(), authorization).execute(make_user(), 5)
    repository.delete.assert_not_awaited()
