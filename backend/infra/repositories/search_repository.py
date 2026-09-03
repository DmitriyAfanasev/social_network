from sqlalchemy import func, or_, select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from backend.application.ports.search_repository import (
    SearchPostView,
    SearchResultView,
    SearchUserView,
    SearchVideoView,
)
from backend.application.ports.search_repository import SearchRepository as SearchRepositoryPort
from backend.infra.models.sqlalchemy import Media, Post, Profile, User, VideoAsset


class SQLAlchemySearchRepository(SearchRepositoryPort):
    def __init__(self, session: AsyncSession) -> None:
        self.session = session

    async def search(self, *, query: str, limit: int) -> SearchResultView:
        pattern = f"%{query.lower()}%"
        users = await self._search_users(pattern=pattern, limit=limit)
        posts = await self._search_posts(pattern=pattern, limit=limit)
        videos = await self._search_videos(pattern=pattern, limit=limit)
        return SearchResultView(users=users, posts=posts, videos=videos)

    async def _search_users(self, *, pattern: str, limit: int) -> tuple[SearchUserView, ...]:
        rows = await self.session.scalars(
            select(User)
            .outerjoin(Profile, Profile.user_id == User.id)
            .options(selectinload(User.profile))
            .where(
                or_(
                    func.lower(User.username).like(pattern),
                    func.lower(Profile.first_name).like(pattern),
                    func.lower(Profile.last_name).like(pattern),
                    func.lower(Profile.city).like(pattern),
                    func.lower(Profile.country).like(pattern),
                )
            )
            .order_by(User.username)
            .limit(limit)
        )
        return tuple(
            SearchUserView(
                id=user.id,
                username=user.username,
                full_name=user.profile.full_name if user.profile else user.username,
                avatar=user.profile.avatar if user.profile else None,
            )
            for user in rows
        )

    async def _search_posts(self, *, pattern: str, limit: int) -> tuple[SearchPostView, ...]:
        rows = await self.session.scalars(
            select(Post)
            .join(User, User.id == Post.author_id)
            .options(selectinload(Post.author).selectinload(User.profile))
            .where(Post.is_published.is_(True), func.lower(Post.content).like(pattern))
            .order_by(Post.created_at.desc(), Post.id.desc())
            .limit(limit)
        )
        return tuple(
            SearchPostView(
                id=post.id,
                content=post.content,
                author_id=post.author_id,
                author_name=(post.author.profile.full_name if post.author and post.author.profile else post.author.username if post.author else "Пользователь"),
                created_at=post.created_at,
            )
            for post in rows
        )

    async def _search_videos(self, *, pattern: str, limit: int) -> tuple[SearchVideoView, ...]:
        rows = (
            await self.session.execute(
                select(VideoAsset, Media, User)
                .join(Media, Media.id == VideoAsset.media_id)
                .outerjoin(User, User.id == Media.uploaded_by)
                .options(selectinload(User.profile))
                .where(
                    Media.deleted_at.is_(None),
                    or_(func.lower(VideoAsset.title).like(pattern), func.lower(Media.original_filename).like(pattern)),
                )
                .order_by(VideoAsset.created_at.desc(), VideoAsset.id.desc())
                .limit(limit)
            )
        ).all()
        return tuple(
            SearchVideoView(
                video_id=video.id,
                media_id=video.media_id,
                title=video.title or media.original_filename or "Видео",
                owner_id=media.uploaded_by,
                owner_name=(owner.profile.full_name if owner and owner.profile else owner.username if owner else "Пользователь"),
                created_at=video.created_at,
            )
            for video, media, owner in rows
        )
