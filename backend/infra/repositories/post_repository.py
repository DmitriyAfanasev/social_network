from collections.abc import Sequence
from typing import Any, cast

from sqlalchemy import delete, func, select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from backend.application.dto import RemovePostImageDTO
from backend.application.ports.post_repository import PostRepository as PostPort
from backend.application.read_models import PostReadModel
from backend.infra.models.sqlalchemy import Comment, LikeComment, LikePost, Media, Post, User


class PostRepository(PostPort):
    def __init__(self, session: AsyncSession) -> None:
        self.session = session

    async def create(
        self,
        content: str | None,
        author_id: int,
        image: str | None,
    ) -> PostReadModel:
        post = Post(content=content, author_id=author_id, image=image)
        self.session.add(post)
        await self.session.flush()
        await self.session.refresh(post)
        cast(Any, post).image_content_type = await self._get_image_content_type(image)
        return cast(PostReadModel, post)

    async def get_by_id(self, post_id: int) -> PostReadModel | None:
        result = await self.session.execute(select(Post).where(Post.id == post_id))
        return cast(PostReadModel | None, result.scalar_one_or_none())

    async def update(
        self,
        post: PostReadModel,
        content: str | None,
        image: str | None,
        author_id: int,
    ) -> PostReadModel:
        post_model = cast(Post, post)
        post_model.content = content
        post_model.image = image
        post_model.author_id = author_id
        await self.session.flush()
        await self.session.refresh(post_model)
        cast(Any, post_model).image_content_type = await self._get_image_content_type(image)
        return cast(PostReadModel, post_model)

    async def delete(self, post: PostReadModel) -> None:
        await self.session.execute(delete(Post).where(Post.id == post.id))
        await self.session.flush()

    async def remove_image(self, post: PostReadModel) -> RemovePostImageDTO:
        if not post.content or not post.content.strip():
            return RemovePostImageDTO(
                success=False,
                message="Нельзя оставлять пост без текста и изображения",
            )

        cast(Post, post).image = None
        await self.session.flush()
        return RemovePostImageDTO(success=True, message="Изображение удалено")

    async def get_paginated_by_likes(
        self,
        page: int = 1,
        limit: int = 10,
        current_user_id: int | None = None,
    ) -> tuple[Sequence[PostReadModel], int]:
        offset = (page - 1) * limit
        statement = (
            select(Post)
            .outerjoin(Post.likes)
            .options(
                selectinload(Post.likes).selectinload(LikePost.user).selectinload(User.profile),
            )
            .group_by(Post.id)
            .order_by(func.count(LikePost.id).desc(), Post.created_at.desc())
            .offset(offset)
            .limit(limit)
        )
        result = await self.session.execute(statement)
        posts = result.scalars().all()

        for post in posts:
            post_view = cast(Any, post)
            post_view.count_comments = await self._count_comments(post.id)
            post_view.preview_comment = await self._get_preview_comment(post.id)
            post_view.image_content_type = await self._get_image_content_type(post.image)
            self._enrich_post_with_likes(post, current_user_id)

        total_count = await self.session.scalar(
            select(func.count(Post.id)).select_from(Post)
        )
        total_pages = ((total_count or 0) + limit - 1) // limit
        return cast(Sequence[PostReadModel], posts), total_pages

    async def get_all_by_author_id(
        self,
        author_id: int,
        current_user_id: int | None = None,
    ) -> Sequence[PostReadModel]:
        statement = (
            select(Post)
            .outerjoin(Post.likes)
            .options(
                selectinload(Post.likes).selectinload(LikePost.user).selectinload(User.profile),
            )
            .where(Post.author_id == author_id)
            .order_by(Post.created_at.desc())
        )
        result = await self.session.execute(statement)
        posts = result.scalars().all()

        for post in posts:
            post_view = cast(Any, post)
            post_view.count_comments = await self._count_comments(post.id)
            post_view.preview_comment = await self._get_preview_comment(post.id)
            post_view.image_content_type = await self._get_image_content_type(post.image)
            self._enrich_post_with_likes(post, current_user_id)
        return cast(Sequence[PostReadModel], posts)

    async def _count_comments(self, post_id: int) -> int:
        return await self.session.scalar(
            select(func.count(Comment.id)).where(Comment.post_id == post_id)
        ) or 0

    async def _get_preview_comment(self, post_id: int) -> Comment | None:
        statement = (
            select(Comment)
            .outerjoin(LikeComment, LikeComment.comment_id == Comment.id)
            .where(Comment.post_id == post_id)
            .options(selectinload(Comment.user).selectinload(User.profile))
            .group_by(Comment.id)
            .order_by(func.count(LikeComment.id).desc(), Comment.created_at.desc(), Comment.id.desc())
            .limit(1)
        )
        comment = await self.session.scalar(statement)
        if comment is not None:
            cast(Any, comment).likes_count = await self.session.scalar(
                select(func.count(LikeComment.id)).where(LikeComment.comment_id == comment.id)
            ) or 0
        return comment

    async def _get_image_content_type(self, image: str | None) -> str | None:
        if not image or not image.startswith("/media/"):
            return None
        media_id = image.removeprefix("/media/").split("/", 1)[0]
        if not media_id.isdigit():
            return None
        return await self.session.scalar(
            select(Media.content_type).where(Media.id == int(media_id))
        )

    @classmethod
    def _enrich_post_with_likes(
        cls,
        post: Post,
        current_user_id: int | None = None,
    ) -> PostReadModel:
        post_view = cast(Any, post)
        post_view.likes_count = len(post.likes)
        post_view.is_liked_by_current = (
            any(like.user_id == current_user_id for like in post.likes)
            if current_user_id
            else False
        )
        post_view.liked_user_ids = {like.user_id for like in post.likes}
        post_view.liked_users = [like.user for like in post.likes]
        return cast(PostReadModel, post)
