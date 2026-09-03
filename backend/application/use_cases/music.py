from pathlib import PurePosixPath

from backend.application.ports.file_upload_service import UploadFileSource
from backend.application.ports.media_storage import MediaStorage
from backend.application.ports.music_repository import MusicRepository, MusicTrackView
from backend.application.ports.transaction_manager import TransactionManager
from backend.domain.user.entity import User


_AUDIO_EXTENSIONS = {".aac", ".flac", ".m4a", ".mp3", ".oga", ".ogg", ".wav", ".webm"}


class ListMusicUseCase:
    def __init__(self, repository: MusicRepository) -> None:
        self.repository = repository

    async def execute(self, *, user_id: int) -> tuple[MusicTrackView, ...]:
        return await self.repository.list_by_user(user_id=user_id)


class UploadMusicUseCase:
    def __init__(
        self,
        media_storage: MediaStorage,
        repository: MusicRepository,
        transaction_manager: TransactionManager,
    ) -> None:
        self.media_storage = media_storage
        self.repository = repository
        self.transaction_manager = transaction_manager

    async def execute(
        self,
        *,
        user: User,
        file: UploadFileSource,
        title: str | None = None,
        artist: str | None = None,
    ) -> MusicTrackView:
        if not self._is_audio(file):
            raise ValueError("Загрузить можно только аудиофайл")

        user_id = user.require_id()
        media = await self.media_storage.upload(
            file=file,
            directory=f"users/{user_id}/music",
            uploaded_by=user_id,
        )
        normalized_title = (title or "").strip()[:200] or media.filename or "Без названия"
        normalized_artist = (artist or "").strip()[:120]
        async with self.transaction_manager:
            return await self.repository.create(
                user_id=user_id,
                media_id=media.id,
                title=normalized_title,
                artist=normalized_artist,
            )

    @staticmethod
    def _is_audio(file: UploadFileSource) -> bool:
        content_type = (file.content_type or "").lower()
        if content_type.startswith("audio/"):
            return True
        filename = file.filename or ""
        return PurePosixPath(filename).suffix.lower() in _AUDIO_EXTENSIONS


class DeleteMusicUseCase:
    def __init__(
        self,
        repository: MusicRepository,
        media_storage: MediaStorage,
        transaction_manager: TransactionManager,
    ) -> None:
        self.repository = repository
        self.media_storage = media_storage
        self.transaction_manager = transaction_manager

    async def execute(self, *, user: User, track_id: int) -> bool:
        async with self.transaction_manager:
            media_id = await self.repository.delete(track_id=track_id, user_id=user.require_id())
            if media_id is None:
                return False
        await self.media_storage.delete(media_id)
        return True
