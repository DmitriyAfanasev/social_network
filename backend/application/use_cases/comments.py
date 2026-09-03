from backend.application.analytics_events import build_analytics_event_payload
from backend.application.commands import CreateCommentCommand, UpdateCommentCommand
from backend.application.event_types import COMMENT_CREATED_EVENT
from backend.application.events import IntegrationEvent
from backend.application.exceptions import NotFoundError, PermissionDeniedError, ValidationAppError
from backend.application.ports.audit_repository import AuditRepository
from backend.application.ports.authorization import AuthorizationService
from backend.application.ports.block_repository import BlockRepository
from backend.application.ports.comment_repository import CommentRepository
from backend.application.ports.outbox_repository import OutboxRepository
from backend.application.ports.transaction_manager import TransactionManager
from backend.application.results import CommentResult, CommentsPageResult
from backend.domain.comment import Comment
from backend.domain.user.entity import User


class CreateCommentUseCase:
    def __init__(
        self,
        comment_repository: CommentRepository,
        outbox_repository: OutboxRepository,
        transaction_manager: TransactionManager,
        block_repository: BlockRepository,
    ) -> None:
        self.comment_repository = comment_repository
        self.outbox_repository = outbox_repository
        self.transaction_manager = transaction_manager
        self.block_repository = block_repository

    async def execute(
        self,
        current_user: User,
        post_id: int,
        command: CreateCommentCommand,
    ) -> CommentResult:
        current_user_id = current_user.require_id()
        post = await self.comment_repository.get_post_by_id(post_id)
        if post is None:
            raise NotFoundError("Post not found")
        if await self.block_repository.is_blocked(current_user_id, post.author_id):
            raise PermissionDeniedError("Нельзя комментировать этот пост")

        if command.parent_id is not None:
            parent_comment = await self.comment_repository.get_comment_by_id(command.parent_id)
            if parent_comment is None:
                raise NotFoundError("Parent comment not found")
            parent_domain_comment = Comment.from_read_model(parent_comment)
            parent_depth = await self.comment_repository.get_comment_depth(command.parent_id)
            if not parent_domain_comment.can_accept_reply_for_post(post_id, parent_depth):
                raise ValidationAppError(
                    "Максимальная глубина дерева комментариев — 3 уровня"
                )

        async with self.transaction_manager:
            comment = await self.comment_repository.create(
                user_id=current_user_id,
                post_id=post_id,
                text=command.content,
                parent_id=command.parent_id,
            )
            await self.outbox_repository.add(
                IntegrationEvent(
                    event_type=COMMENT_CREATED_EVENT,
                    payload=build_analytics_event_payload(
                        user_id=current_user_id,
                        entity_type="comment",
                        entity_id=comment.id,
                        data={
                            "post_id": post_id,
                            "parent_id": command.parent_id,
                        },
                    ),
                )
            )
        return CommentResult(comment=comment, author=current_user)


class GetCommentsUseCase:
    def __init__(self, comment_repository: CommentRepository) -> None:
        self.comment_repository = comment_repository

    async def execute(
        self,
        post_id: int,
        offset: int = 0,
        limit: int = 5,
    ) -> CommentsPageResult:
        comments, has_more = await self.comment_repository.get_paginated(
            post_id,
            offset,
            limit,
        )
        return CommentsPageResult(
            comments=comments,
            has_more=has_more,
            offset=offset + limit,
            limit=limit,
            post_id=post_id,
        )


class UpdateCommentUseCase:
    def __init__(
        self,
        comment_repository: CommentRepository,
        transaction_manager: TransactionManager,
    ) -> None:
        self.comment_repository = comment_repository
        self.transaction_manager = transaction_manager

    async def execute(
        self,
        current_user: User,
        comment_id: int,
        command: UpdateCommentCommand,
    ) -> CommentResult:
        text = command.content.strip()
        if not text:
            raise ValidationAppError("Напишите комментарий.")

        comment = await self.comment_repository.get_comment_by_id(comment_id)
        if comment is None:
            raise NotFoundError("Комментарий не найден")

        current_user_id = current_user.require_id()
        if comment.user_id != current_user_id:
            raise PermissionDeniedError("У вас нет прав на редактирование этого комментария")

        async with self.transaction_manager:
            updated_comment = await self.comment_repository.update(comment, text)

        return CommentResult(comment=updated_comment, author=current_user)


class DeleteCommentUseCase:
    def __init__(
        self,
        comment_repository: CommentRepository,
        transaction_manager: TransactionManager,
        authorization: AuthorizationService | None = None,
        audit_repository: AuditRepository | None = None,
    ) -> None:
        self.comment_repository = comment_repository
        self.transaction_manager = transaction_manager
        self.authorization = authorization
        self.audit_repository = audit_repository

    async def execute(self, current_user: User, comment_id: int) -> None:
        comment = await self.comment_repository.get_comment_by_id(comment_id)
        if comment is None:
            raise NotFoundError("Комментарий не найден")

        current_user_id = current_user.require_id()
        post = await self.comment_repository.get_post_by_id(comment.post_id)
        is_post_owner = post is not None and post.author_id == current_user_id
        is_moderator = self.authorization is not None and await self.authorization.has_permission(
            current_user_id, "comments.delete_any"
        )
        if comment.user_id != current_user_id and not is_post_owner and not is_moderator:
            raise PermissionDeniedError("У вас нет прав на удаление этого комментария")

        async with self.transaction_manager:
            await self.comment_repository.delete(comment)
            if is_moderator and self.audit_repository:
                await self.audit_repository.record(
                    current_user_id,
                    "comment.deleted_by_moderator",
                    "comment",
                    comment_id,
                    {"post_id": comment.post_id, "comment_author_id": comment.user_id},
                )
