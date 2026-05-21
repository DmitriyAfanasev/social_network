from typing import cast

from backend.application.analytics_events import build_analytics_event_payload
from backend.application.commands import CreateCommentCommand
from backend.application.event_types import COMMENT_CREATED_EVENT
from backend.application.events import IntegrationEvent
from backend.application.exceptions import NotFoundError, ValidationAppError
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
    ) -> None:
        self.comment_repository = comment_repository
        self.outbox_repository = outbox_repository
        self.transaction_manager = transaction_manager

    async def execute(
        self,
        current_user: User,
        post_id: int,
        command: CreateCommentCommand,
    ) -> CommentResult:
        current_user_id = cast(int, current_user.id)
        post = await self.comment_repository.get_post_by_id(post_id)
        if post is None:
            raise NotFoundError("Post not found")

        if command.parent_id is not None:
            parent_comment = await self.comment_repository.get_comment_by_id(command.parent_id)
            if parent_comment is None:
                raise NotFoundError("Parent comment not found")
            parent_domain_comment = Comment.from_read_model(parent_comment)
            if not parent_domain_comment.can_accept_reply_for_post(post_id):
                raise ValidationAppError(
                    "Parent comment must be a direct comment for this post"
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
