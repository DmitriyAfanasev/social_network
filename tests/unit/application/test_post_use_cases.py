from datetime import UTC, datetime
from types import SimpleNamespace
from unittest.mock import AsyncMock

import pytest

from backend.application.commands import CreatePostCommand, UpdatePostCommand
from backend.application.exceptions import (
    ExternalServiceError,
    NotFoundError,
    PermissionDeniedError,
    ValidationAppError,
)
from backend.application.use_cases.posts import (
    CreatePostUseCase,
    DeletePostUseCase,
    GetFeedUseCase,
    RemovePostImageUseCase,
    UpdatePostUseCase,
)
from backend.domain.user.entity import User
from tests.fakes import FakeTransaction


def make_user(user_id: int = 1, *, superuser: bool = False) -> User:
    return User(
        id=user_id,
        username=f"user{user_id}",
        email=f"user{user_id}@example.com",
        hashed_password="hash",
        is_superuser=superuser,
    )


def make_post(
    *, author_id: int = 1, content: str | None = "text", image: str | None = None
) -> SimpleNamespace:
    now = datetime.now(UTC)
    return SimpleNamespace(
        id=10,
        author_id=author_id,
        content=content,
        image=image,
        created_at=now,
        updated_at=now,
    )


def make_image(filename: str = "photo.png") -> SimpleNamespace:
    return SimpleNamespace(filename=filename)


@pytest.mark.application
@pytest.mark.asyncio
async def test_feed_passes_optional_current_user_to_repository() -> None:
    repository = AsyncMock()
    repository.get_paginated_by_likes.return_value = (["post"], 3)
    user = make_user()

    result = await GetFeedUseCase(repository).execute(user, page=2)

    assert result.posts == ["post"]
    assert result.page == 2
    repository.get_paginated_by_likes.assert_awaited_once_with(page=2, current_user_id=1)


@pytest.mark.application
@pytest.mark.asyncio
async def test_create_post_without_text_or_attachment_is_rejected() -> None:
    repository = AsyncMock()
    use_case = CreatePostUseCase(repository, AsyncMock(), FakeTransaction(), AsyncMock())

    with pytest.raises(ValidationAppError):
        await use_case.execute(make_user(), CreatePostCommand(content="  "))

    repository.create.assert_not_awaited()


@pytest.mark.application
@pytest.mark.asyncio
async def test_create_post_uploads_attachment_and_emits_event() -> None:
    repository = AsyncMock()
    created = make_post(content=None)
    updated = make_post(content=None, image="/files/photo.png")
    repository.create.return_value = created
    repository.update.return_value = updated
    outbox = AsyncMock()
    upload = AsyncMock(return_value="/files/photo.png")

    image = make_image()
    result = await CreatePostUseCase(
        repository, outbox, FakeTransaction(), SimpleNamespace(upload=upload)
    ).execute(make_user(), CreatePostCommand(content=None, image=image))

    assert result.post is updated
    upload.assert_awaited_once_with(file=image, directory="posts/10/attachments")
    repository.update.assert_awaited_once()
    outbox.add.assert_awaited_once()


@pytest.mark.application
@pytest.mark.asyncio
async def test_create_post_converts_upload_failures_to_application_errors() -> None:
    repository = AsyncMock()
    repository.create.return_value = make_post(content=None)
    upload = AsyncMock(side_effect=RuntimeError("storage down"))

    with pytest.raises(ExternalServiceError):
        await CreatePostUseCase(
            repository, AsyncMock(), FakeTransaction(), SimpleNamespace(upload=upload)
        ).execute(make_user(), CreatePostCommand(content=None, image=make_image()))


@pytest.mark.application
@pytest.mark.asyncio
@pytest.mark.parametrize("owner", [False, True])
async def test_update_post_allows_owner_or_superuser_only(owner: bool) -> None:
    repository = AsyncMock()
    repository.get_by_id.return_value = make_post(author_id=1)
    use_case = UpdatePostUseCase(repository, FakeTransaction(), AsyncMock())
    user = make_user(1 if owner else 2, superuser=not owner)

    await use_case.execute(user, 10, UpdatePostCommand(content="changed"))
    repository.update.assert_awaited_once()


@pytest.mark.application
@pytest.mark.asyncio
async def test_update_post_rejects_other_user() -> None:
    repository = AsyncMock()
    repository.get_by_id.return_value = make_post(author_id=1)

    with pytest.raises(PermissionDeniedError):
        await UpdatePostUseCase(repository, FakeTransaction(), AsyncMock()).execute(
            make_user(2), 10, UpdatePostCommand(content="changed")
        )

    repository.update.assert_not_awaited()


@pytest.mark.application
@pytest.mark.asyncio
async def test_delete_post_not_found_and_success_paths() -> None:
    repository = AsyncMock()
    use_case = DeletePostUseCase(repository, FakeTransaction())
    repository.get_by_id.return_value = None
    with pytest.raises(NotFoundError):
        await use_case.execute(make_user(), 10)

    repository.get_by_id.return_value = make_post()
    result = await use_case.execute(make_user(), 10)
    assert result.message == "Пост успешно удален"
    repository.delete.assert_awaited_once()


@pytest.mark.application
@pytest.mark.asyncio
async def test_remove_post_image_requires_remaining_text_and_reports_result() -> None:
    repository = AsyncMock()
    repository.get_by_id.return_value = make_post(content=None, image="image")
    use_case = RemovePostImageUseCase(repository, FakeTransaction())

    with pytest.raises(ValidationAppError):
        await use_case.execute(make_user(), 10)
    repository.remove_image.assert_not_awaited()

    repository.get_by_id.return_value = make_post(content="text", image="image")
    repository.remove_image.return_value = SimpleNamespace(success=True, message="removed")
    result = await use_case.execute(make_user(), 10)
    assert result.message == "removed"
