from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from backend.application.dto import ToggleLikeDTO
from backend.application.ports.like_repository import LikeRepository as LikePort
from backend.infra.models.sqlalchemy import LikeComment, LikePost


class LikeRepository(LikePort):
    def __init__(self, session: AsyncSession) -> None:
        self.session = session

    async def toggle_post_like(self, user_id: int, post_id: int) -> ToggleLikeDTO:
        statement = select(LikePost).where(
            LikePost.user_id == user_id,
            LikePost.post_id == post_id,
        )
        result = await self.session.execute(statement)
        like = result.scalars().first()

        if like:
            await self.session.delete(like)
            action = "removed"
        else:
            self.session.add(LikePost(post_id=post_id, user_id=user_id))
            action = "added"

        await self.session.flush()

        count_statement = select(func.count()).where(LikePost.post_id == post_id)
        likes_count = (await self.session.execute(count_statement)).scalar_one_or_none() or 0
        return ToggleLikeDTO(
            success=True,
            action=action,
            likes_count=likes_count,
            liked=action == "added",
        )

    async def toggle_comment_like(self, user_id: int, comment_id: int) -> ToggleLikeDTO:
        result = await self.session.execute(select(LikeComment).where(
            LikeComment.user_id == user_id,
            LikeComment.comment_id == comment_id,
        ))
        like = result.scalar_one_or_none()
        if like:
            await self.session.delete(like)
            action = "removed"
        else:
            self.session.add(LikeComment(user_id=user_id, comment_id=comment_id))
            action = "added"
        await self.session.flush()
        likes_count = await self.session.scalar(
            select(func.count()).where(LikeComment.comment_id == comment_id)
        ) or 0
        return ToggleLikeDTO(success=True, action=action, likes_count=likes_count, liked=action == "added")
