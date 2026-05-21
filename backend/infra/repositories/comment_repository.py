from typing import cast

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from backend.application.ports.comment_repository import CommentRepository as CommentPort
from backend.application.read_models import CommentReadModel, PostReadModel
from backend.infra.models.sqlalchemy import Comment, Post, User


class CommentRepository(CommentPort):
    def __init__(self, session: AsyncSession) -> None:
        self.session = session

    async def get_post_by_id(self, post_id: int) -> PostReadModel | None:
        return cast(PostReadModel | None, await self.session.get(Post, post_id))

    async def get_comment_by_id(self, comment_id: int) -> CommentReadModel | None:
        return cast(CommentReadModel | None, await self.session.get(Comment, comment_id))

    async def create(
        self,
        user_id: int,
        post_id: int,
        text: str,
        parent_id: int | None = None,
    ) -> CommentReadModel:
        comment = Comment(
            user_id=user_id,
            post_id=post_id,
            parent_id=parent_id,
            text=text,
        )
        self.session.add(comment)
        await self.session.flush()
        await self.session.refresh(comment)
        return cast(CommentReadModel, comment)

    async def get_paginated(
        self,
        post_id: int,
        offset: int,
        limit: int,
    ) -> tuple[list[CommentReadModel], bool]:
        statement = (
            select(Comment)
            .where(Comment.post_id == post_id)
            .options(selectinload(Comment.user).selectinload(User.profile))
            .order_by(Comment.created_at.asc())
            .offset(offset)
            .limit(limit)
        )
        result = await self.session.execute(statement)
        comments = cast(list[CommentReadModel], list(result.scalars().all()))

        has_more_statement = (
            select(Comment.id)
            .where(Comment.post_id == post_id)
            .offset(offset + limit)
            .limit(1)
        )
        has_more_result = await self.session.execute(has_more_statement)
        has_more = has_more_result.scalars().first() is not None
        return comments, has_more
