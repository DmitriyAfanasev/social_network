from datetime import UTC, datetime

from sqlalchemy import delete, func, or_, select, update
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from backend.application.ports.video_repository import (
    VideoAlbumView,
    VideoRenditionInput,
    VideoRenditionView,
    VideoRepository,
    VideoView,
)
from backend.infra.models.sqlalchemy.media import Media
from backend.infra.models.sqlalchemy.user import User
from backend.infra.models.sqlalchemy.video import (
    VideoAlbum,
    VideoAsset,
    VideoBookmark,
    VideoFavorite,
    VideoLike,
    VideoRendition,
    VideoViewRecord,
)


class SQLAlchemyVideoRepository(VideoRepository):
    def __init__(self, session: AsyncSession) -> None:
        self.session = session

    async def get_owner_name(self, *, user_id: int) -> str | None:
        owner = await self.session.scalar(select(User).options(selectinload(User.profile)).where(User.id == user_id))
        if owner is None:
            return None
        if owner.profile and owner.profile.first_name and owner.profile.last_name:
            return f"{owner.profile.first_name} {owner.profile.last_name}"
        return owner.username

    async def source_object_key(self, *, media_id: int) -> str:
        object_key = await self.session.scalar(select(Media.object_key).where(Media.id == media_id))
        if object_key is None:
            raise RuntimeError(f"Media {media_id} was not found after upload")
        return object_key

    async def list_albums(
        self,
        *,
        user_id: int,
        viewer_id: int | None = None,
        tab: str = "uploaded",
        search: str = "",
    ) -> tuple[VideoAlbumView, ...]:
        statement = (
            select(VideoAsset, Media)
            .join(Media, Media.id == VideoAsset.media_id)
            .where(Media.deleted_at.is_(None))
            .options(selectinload(VideoAsset.album))
            .order_by(VideoAsset.created_at.desc(), VideoAsset.id.desc())
        )
        if tab == "bookmarked":
            statement = statement.join(VideoBookmark, VideoBookmark.video_id == VideoAsset.id).where(
                VideoBookmark.user_id == user_id
            )
        elif tab == "favorite":
            statement = statement.join(VideoFavorite, VideoFavorite.video_id == VideoAsset.id).where(
                VideoFavorite.user_id == user_id
            )
        elif tab == "viewed":
            statement = statement.join(VideoViewRecord, VideoViewRecord.video_id == VideoAsset.id).where(
                VideoViewRecord.user_id == user_id
            )
        else:
            statement = statement.where(Media.uploaded_by == user_id)

        normalized_search = search.strip().lower()
        if normalized_search:
            pattern = f"%{normalized_search}%"
            statement = statement.where(
                or_(func.lower(VideoAsset.title).like(pattern), func.lower(Media.original_filename).like(pattern))
            )

        rows = (await self.session.execute(statement)).all()
        grouped: dict[int, list[VideoView]] = {}
        titles: dict[int, str] = {}
        for asset, media in rows:
            video = await self._to_view(asset, media, viewer_id)
            group_id = 0 if tab in {"bookmarked", "favorite", "viewed"} else asset.album_id or 0
            grouped.setdefault(group_id, []).append(video)
            titles[group_id] = {
                "bookmarked": "Смотреть позже",
                "favorite": "Избранное",
                "viewed": "Просмотренные",
            }.get(tab, asset.album.title if asset.album else "Без альбома")

        return tuple(VideoAlbumView(group_id, titles[group_id], tuple(videos)) for group_id, videos in grouped.items())

    async def create_album(self, *, user_id: int, title: str) -> int:
        album = VideoAlbum(user_id=user_id, title=title)
        self.session.add(album)
        await self.session.flush()
        return album.id

    async def album_belongs_to_user(self, *, album_id: int, user_id: int) -> bool:
        return (
            await self.session.scalar(
                select(VideoAlbum.id).where(VideoAlbum.id == album_id, VideoAlbum.user_id == user_id)
            )
        ) is not None

    async def delete_video(self, *, video_id: int, user_id: int) -> int | None:
        media_id = await self.session.scalar(
            select(VideoAsset.media_id)
            .join(Media, Media.id == VideoAsset.media_id)
            .where(VideoAsset.id == video_id, Media.uploaded_by == user_id, Media.deleted_at.is_(None))
        )
        if media_id is None:
            return None
        await self.session.execute(delete(VideoAsset).where(VideoAsset.id == video_id))
        return media_id

    async def delete_album(self, *, album_id: int, user_id: int) -> tuple[int, ...] | None:
        owner = await self.session.scalar(select(VideoAlbum.id).where(VideoAlbum.id == album_id, VideoAlbum.user_id == user_id))
        if owner is None:
            return None
        media_ids = tuple((await self.session.scalars(select(VideoAsset.media_id).where(VideoAsset.album_id == album_id))).all())
        await self.session.execute(delete(VideoAsset).where(VideoAsset.album_id == album_id))
        await self.session.execute(delete(VideoAlbum).where(VideoAlbum.id == album_id))
        return media_ids

    async def create(self, *, media_id: int, album_id: int | None = None, title: str = "Видео") -> int:
        asset = VideoAsset(media_id=media_id, album_id=album_id, title=title, status="uploaded")
        self.session.add(asset)
        await self.session.flush()
        return asset.id

    async def get_for_user(self, *, video_id: int, user_id: int | None = None) -> VideoView | None:
        row = (
            await self.session.execute(
                select(VideoAsset, Media)
                .join(Media, Media.id == VideoAsset.media_id)
                .where(VideoAsset.id == video_id, Media.deleted_at.is_(None))
                .options(selectinload(VideoAsset.album))
            )
        ).first()
        if row is None:
            return None
        asset, media = row
        return await self._to_view(asset, media, user_id)

    async def record_view(self, *, video_id: int, user_id: int | None = None) -> int | None:
        exists = await self.session.scalar(
            select(VideoAsset.id)
            .join(Media, Media.id == VideoAsset.media_id)
            .where(VideoAsset.id == video_id, Media.deleted_at.is_(None))
        )
        if exists is None:
            return None
        if user_id is not None:
            already_recorded = await self.session.scalar(
                select(VideoViewRecord.id).where(
                    VideoViewRecord.video_id == video_id,
                    VideoViewRecord.user_id == user_id,
                )
            )
            if already_recorded is None:
                self.session.add(VideoViewRecord(video_id=video_id, user_id=user_id))
                await self.session.flush()
        return int(await self.session.scalar(select(func.count(VideoViewRecord.id)).where(VideoViewRecord.video_id == video_id)) or 0)

    async def toggle_like(self, *, video_id: int, user_id: int) -> tuple[int, bool] | None:
        if await self.session.scalar(select(VideoAsset.id).where(VideoAsset.id == video_id)) is None:
            return None
        like = await self.session.scalar(
            select(VideoLike).where(VideoLike.video_id == video_id, VideoLike.user_id == user_id)
        )
        if like is None:
            self.session.add(VideoLike(video_id=video_id, user_id=user_id))
            liked = True
        else:
            await self.session.delete(like)
            liked = False
        await self.session.flush()
        count = int(await self.session.scalar(select(func.count(VideoLike.id)).where(VideoLike.video_id == video_id)) or 0)
        return count, liked

    async def add_bookmark(self, *, video_id: int, user_id: int) -> bool | None:
        if await self.session.scalar(select(VideoAsset.id).where(VideoAsset.id == video_id)) is None:
            return None
        bookmark = await self.session.scalar(
            select(VideoBookmark).where(VideoBookmark.video_id == video_id, VideoBookmark.user_id == user_id)
        )
        if bookmark is None:
            self.session.add(VideoBookmark(video_id=video_id, user_id=user_id))
        await self.session.flush()
        return True

    async def remove_bookmark(self, *, video_id: int, user_id: int) -> bool | None:
        if await self.session.scalar(select(VideoAsset.id).where(VideoAsset.id == video_id)) is None:
            return None
        bookmark = await self.session.scalar(
            select(VideoBookmark).where(VideoBookmark.video_id == video_id, VideoBookmark.user_id == user_id)
        )
        if bookmark is not None:
            await self.session.delete(bookmark)
        await self.session.flush()
        return False

    async def complete_transcoding(
        self,
        *,
        video_id: int,
        status: str,
        renditions: tuple[VideoRenditionInput, ...] = (),
        duration: float | None = None,
        error_message: str | None = None,
    ) -> bool:
        asset = await self.session.get(VideoAsset, video_id)
        if asset is None:
            return False
        asset.status = status
        asset.error_message = error_message
        asset.processed_at = datetime.now(UTC) if status in {"ready", "failed"} else None
        if duration is not None:
            await self.session.execute(update(Media).where(Media.id == asset.media_id).values(duration=duration))
        if status == "ready":
            await self.session.execute(delete(VideoRendition).where(VideoRendition.video_id == video_id))
            for rendition in renditions:
                self.session.add(
                    VideoRendition(
                        video_id=video_id,
                        height=rendition.height,
                        object_key=rendition.object_key,
                        content_type=rendition.content_type,
                        size=rendition.size,
                        width=rendition.width,
                        duration=rendition.duration,
                    )
                )
        await self.session.flush()
        return True

    async def _to_view(self, asset: VideoAsset, media: Media, viewer_id: int | None) -> VideoView:
        owner = None
        if media.uploaded_by is not None:
            owner = await self.session.scalar(
                select(User).options(selectinload(User.profile)).where(User.id == media.uploaded_by)
            )
        renditions = await self.session.scalars(
            select(VideoRendition).where(VideoRendition.video_id == asset.id).order_by(VideoRendition.height)
        )
        likes_count = int(await self.session.scalar(select(func.count(VideoLike.id)).where(VideoLike.video_id == asset.id)) or 0)
        views_count = int(await self.session.scalar(select(func.count(VideoViewRecord.id)).where(VideoViewRecord.video_id == asset.id)) or 0)
        is_liked = False
        is_bookmarked = False
        is_favorited = False
        if viewer_id is not None:
            is_liked = (
                await self.session.scalar(
                    select(VideoLike.id).where(VideoLike.video_id == asset.id, VideoLike.user_id == viewer_id)
                )
            ) is not None
            is_bookmarked = (
                await self.session.scalar(
                    select(VideoBookmark.id).where(VideoBookmark.video_id == asset.id, VideoBookmark.user_id == viewer_id)
                )
            ) is not None
            is_favorited = (
                await self.session.scalar(
                    select(VideoFavorite.id).where(VideoFavorite.video_id == asset.id, VideoFavorite.user_id == viewer_id)
                )
            ) is not None
        return VideoView(
            id=asset.id,
            media_id=asset.media_id,
            status=asset.status,
            error_message=asset.error_message,
            renditions=tuple(
                VideoRenditionView(r.height, r.object_key, r.content_type, r.size) for r in renditions
            ),
            original_filename=media.original_filename,
            created_at=media.created_at,
            duration=media.duration,
            title=asset.title or media.original_filename or "Видео",
            owner_id=media.uploaded_by,
            owner_name=(
                f"{owner.profile.first_name} {owner.profile.last_name}"
                if owner and owner.profile and owner.profile.first_name and owner.profile.last_name
                else owner.username if owner else "пользователя"
            ),
            views_count=views_count,
            likes_count=likes_count,
            is_liked_by_current=is_liked,
            is_bookmarked_by_current=is_bookmarked,
            is_favorited_by_current=is_favorited,
        )

    async def add_favorite(self, *, video_id: int, user_id: int) -> bool | None:
        if await self.session.scalar(select(VideoAsset.id).where(VideoAsset.id == video_id)) is None:
            return None
        favorite = await self.session.scalar(
            select(VideoFavorite).where(VideoFavorite.video_id == video_id, VideoFavorite.user_id == user_id)
        )
        if favorite is None:
            self.session.add(VideoFavorite(video_id=video_id, user_id=user_id))
        await self.session.flush()
        return True

    async def remove_favorite(self, *, video_id: int, user_id: int) -> bool | None:
        if await self.session.scalar(select(VideoAsset.id).where(VideoAsset.id == video_id)) is None:
            return None
        favorite = await self.session.scalar(
            select(VideoFavorite).where(VideoFavorite.video_id == video_id, VideoFavorite.user_id == user_id)
        )
        if favorite is not None:
            await self.session.delete(favorite)
        await self.session.flush()
        return False
