from sqlalchemy import delete, select
from sqlalchemy.ext.asyncio import AsyncSession

from backend.application.ports.music_repository import MusicRepository as MusicRepositoryPort
from backend.application.ports.music_repository import MusicTrackView
from backend.infra.models.sqlalchemy import Media, MusicTrack


class SQLAlchemyMusicRepository(MusicRepositoryPort):
    def __init__(self, session: AsyncSession) -> None:
        self.session = session

    async def list_by_user(self, *, user_id: int) -> tuple[MusicTrackView, ...]:
        rows = (
            await self.session.execute(
                select(MusicTrack, Media)
                .join(Media, Media.id == MusicTrack.media_id)
                .where(MusicTrack.user_id == user_id, Media.deleted_at.is_(None))
                .order_by(MusicTrack.created_at.desc(), MusicTrack.id.desc())
            )
        ).all()
        return tuple(self._to_view(track, media) for track, media in rows)

    async def create(
        self,
        *,
        user_id: int,
        media_id: int,
        title: str,
        artist: str,
    ) -> MusicTrackView:
        track = MusicTrack(user_id=user_id, media_id=media_id, title=title, artist=artist)
        self.session.add(track)
        await self.session.flush()
        media = await self.session.get(Media, media_id)
        if media is None:
            raise RuntimeError("Медиафайл не найден после загрузки")
        return self._to_view(track, media)

    async def delete(self, *, track_id: int, user_id: int) -> int | None:
        media_id = await self.session.scalar(
            select(MusicTrack.media_id).where(MusicTrack.id == track_id, MusicTrack.user_id == user_id)
        )
        if media_id is None:
            return None
        await self.session.execute(
            delete(MusicTrack).where(MusicTrack.id == track_id, MusicTrack.user_id == user_id)
        )
        return media_id

    @staticmethod
    def _to_view(track: MusicTrack, media: Media) -> MusicTrackView:
        return MusicTrackView(
            id=track.id,
            media_id=track.media_id,
            user_id=track.user_id,
            title=track.title,
            artist=track.artist,
            duration=media.duration,
            created_at=track.created_at,
        )
