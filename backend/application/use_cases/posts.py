import logging

from backend.application.analytics_events import build_analytics_event_payload
from backend.application.commands import CreatePostCommand, UpdatePostCommand
from backend.application.event_types import POST_CREATED_EVENT
from backend.application.events import IntegrationEvent
from backend.application.exceptions import (
    ExternalServiceError,
    NotFoundError,
    PermissionDeniedError,
    ValidationAppError,
)
from backend.application.ports.file_upload_service import FileUploadService
from backend.application.ports.outbox_repository import OutboxRepository
from backend.application.ports.post_repository import PostRepository
from backend.application.ports.transaction_manager import TransactionManager
from backend.application.results import (
    FeedResult,
    MessageResult,
    PostResult,
    RemovePostImageResult,
)
from backend.domain.post import Post, PostDraft
from backend.domain.user.entity import User


logger = logging.getLogger(__name__)


def _normalize_post_content(content: str | None, *, has_attachment: bool) -> str | None:
    try:
        return PostDraft.from_input(
            content,
            has_attachment=has_attachment,
        ).content
    except ValueError as e:
        raise ValidationAppError("Добавьте текст или файл.") from e


def _post_attachment_directory(post_id: int) -> str:
    return f"posts/{post_id}/attachments"


class GetFeedUseCase:
    def __init__(self, post_repository: PostRepository) -> None:
        self.post_repository = post_repository

    async def execute(self, current_user: User | None, page: int = 1) -> FeedResult:
        posts, total_pages = await self.post_repository.get_paginated_by_likes(
            page=page,
            current_user_id=current_user.require_id() if current_user is not None else None,
        )
        return FeedResult(
            current_user=current_user,
            posts=posts,
            page=page,
            total_pages=total_pages,
        )


class CreatePostUseCase:
    def __init__(
        self,
        post_repository: PostRepository,
        outbox_repository: OutboxRepository,
        transaction_manager: TransactionManager,
        file_upload_service: FileUploadService,
    ) -> None:
        self.post_repository = post_repository
        self.outbox_repository = outbox_repository
        self.transaction_manager = transaction_manager
        self.file_upload_service = file_upload_service

    async def execute(self, current_user: User, command: CreatePostCommand) -> PostResult:
        current_user_id = current_user.require_id()
        has_attachment = bool(command.image and command.image.filename)
        content = _normalize_post_content(command.content, has_attachment=has_attachment)

        async with self.transaction_manager:
            new_post = await self.post_repository.create(
                content=content,
                author_id=current_user_id,
                image=None,
            )

            if has_attachment and command.image:
                try:
                    image_url = await self.file_upload_service.upload(
                        file=command.image,
                        directory=_post_attachment_directory(new_post.id),
                    )
                except ValueError as e:
                    raise ValidationAppError(str(e)) from e
                except (OSError, RuntimeError) as e:
                    logger.exception("Post attachment upload failed: %s", e)
                    raise ExternalServiceError("Ошибка при обработке файла") from e

                new_post = await self.post_repository.update(
                    post=new_post,
                    content=content,
                    image=image_url,
                    author_id=current_user_id,
                )
            await self.outbox_repository.add(
                IntegrationEvent(
                    event_type=POST_CREATED_EVENT,
                    payload=build_analytics_event_payload(
                        user_id=current_user_id,
                        entity_type="post",
                        entity_id=new_post.id,
                        data={
                            "has_image": bool(new_post.image),
                        },
                    ),
                )
            )
        return PostResult(post=new_post, author=current_user)


class UpdatePostUseCase:
    def __init__(
        self,
        post_repository: PostRepository,
        transaction_manager: TransactionManager,
        file_upload_service: FileUploadService,
    ) -> None:
        self.post_repository = post_repository
        self.transaction_manager = transaction_manager
        self.file_upload_service = file_upload_service

    async def execute(
        self,
        current_user: User,
        post_id: int,
        command: UpdatePostCommand,
    ) -> PostResult:
        current_user_id = current_user.require_id()
        post = await self.post_repository.get_by_id(post_id)
        if not post:
            raise NotFoundError("Пост не найден")

        domain_post = Post.from_read_model(post)
        if not domain_post.can_be_managed_by(current_user):
            raise PermissionDeniedError("У вас нет прав на редактирование этого поста")

        image_url = post.image
        has_new_attachment = bool(command.image and command.image.filename)
        content = _normalize_post_content(
            command.content,
            has_attachment=has_new_attachment or bool(image_url),
        )

        if has_new_attachment and command.image:
            try:
                image_url = await self.file_upload_service.upload(
                    file=command.image,
                    directory=_post_attachment_directory(post_id),
                )
            except ValueError as e:
                raise ValidationAppError(str(e)) from e
            except (OSError, RuntimeError) as e:
                logger.exception("Post attachment upload failed: %s", e)
                raise ExternalServiceError("Ошибка при обработке файла") from e

        try:
            async with self.transaction_manager:
                updated_post = await self.post_repository.update(
                    post=post,
                    content=content,
                    image=image_url,
                    author_id=current_user_id,
                )
        except (OSError, RuntimeError) as e:
            logger.exception("Post update failed: %s", e)
            raise ExternalServiceError("Не удалось обновить пост") from e

        return PostResult(post=updated_post)


class DeletePostUseCase:
    def __init__(
        self,
        post_repository: PostRepository,
        transaction_manager: TransactionManager,
    ) -> None:
        self.post_repository = post_repository
        self.transaction_manager = transaction_manager

    async def execute(self, current_user: User, post_id: int) -> MessageResult:
        post = await self.post_repository.get_by_id(post_id)
        if not post:
            raise NotFoundError("Пост не найден")

        domain_post = Post.from_read_model(post)
        if not domain_post.can_be_managed_by(current_user):
            raise PermissionDeniedError("У вас нет прав на удаление этого поста")

        async with self.transaction_manager:
            await self.post_repository.delete(post=post)
        return MessageResult("Пост успешно удален")


class RemovePostImageUseCase:
    def __init__(
        self,
        post_repository: PostRepository,
        transaction_manager: TransactionManager,
    ) -> None:
        self.post_repository = post_repository
        self.transaction_manager = transaction_manager

    async def execute(self, current_user: User, post_id: int) -> RemovePostImageResult:
        post = await self.post_repository.get_by_id(post_id)
        if not post:
            raise NotFoundError("Пост не найден или недостаточно прав")

        domain_post = Post.from_read_model(post)
        if not domain_post.can_be_managed_by(current_user):
            raise NotFoundError("Пост не найден или недостаточно прав")
        if not domain_post.can_remove_image():
            raise ValidationAppError("Нельзя оставлять пост без текста и изображения")

        async with self.transaction_manager:
            result = await self.post_repository.remove_image(post)
        if not result.success:
            raise ValidationAppError(result.message)
        return RemovePostImageResult(
            success=result.success,
            message=result.message,
        )
