from dataclasses import dataclass

from backend.application.events import IntegrationEvent
from backend.application.ports.file_upload_service import UploadFileSource
from backend.application.ports.media_storage import MediaStorage
from backend.application.ports.outbox_repository import OutboxRepository
from backend.application.ports.transaction_manager import TransactionManager
from backend.application.ports.video_repository import VideoAlbumView, VideoRepository, VideoView
from backend.domain.user.entity import User


VIDEO_TRANSCODE_REQUESTED = "video.transcode.requested"


@dataclass(frozen=True, slots=True)
class UploadVideoResult:
    video_id: int
    media_id: int
    status: str


class UploadVideoUseCase:
    def __init__(
        self,
        media_storage: MediaStorage,
        video_repository: VideoRepository,
        outbox_repository: OutboxRepository,
        transaction_manager: TransactionManager,
    ) -> None:
        self.media_storage = media_storage
        self.video_repository = video_repository
        self.outbox_repository = outbox_repository
        self.transaction_manager = transaction_manager

    async def execute(
        self,
        *,
        user: User,
        file: UploadFileSource,
        album_id: int | None = None,
        title: str | None = None,
    ) -> UploadVideoResult:
        if not file.content_type or not file.content_type.startswith("video/"):
            raise ValueError("Загрузить можно только видеофайл")
        user_id = user.require_id()
        if album_id is not None and not await self.video_repository.album_belongs_to_user(album_id=album_id, user_id=user_id):
            raise ValueError("Видеоальбом не найден")
        media = await self.media_storage.upload(file=file, directory="videos/originals", uploaded_by=user_id)
        object_key = await self.video_repository.source_object_key(media_id=media.id)
        normalized_title = (title or "").strip()[:200] or media.filename or "Видео"
        video_id = await self.video_repository.create(media_id=media.id, album_id=album_id, title=normalized_title)
        await self.outbox_repository.add(
            IntegrationEvent(
                event_type=VIDEO_TRANSCODE_REQUESTED,
                payload={
                    "video_id": video_id,
                    "media_id": media.id,
                    "bucket": "",
                    "object_key": object_key,
                    "requested_heights": [360, 720, 1080],
                },
            )
        )
        async with self.transaction_manager:
            pass
        return UploadVideoResult(video_id=video_id, media_id=media.id, status="uploaded")


class GetVideoUseCase:
    def __init__(self, repository: VideoRepository) -> None:
        self.repository = repository

    async def execute(self, *, video_id: int, user_id: int | None = None) -> VideoView | None:
        return await self.repository.get_for_user(video_id=video_id, user_id=user_id)


class ListVideoAlbumsUseCase:
    def __init__(self, repository: VideoRepository) -> None:
        self.repository = repository

    async def get_owner_name(self, *, user_id: int) -> str:
        return await self.repository.get_owner_name(user_id=user_id) or "пользователя"

    async def execute(
        self,
        *,
        user: User,
        viewer_id: int | None = None,
        tab: str = "uploaded",
        search: str = "",
    ) -> tuple[VideoAlbumView, ...]:
        user_id = user.require_id()
        return await self.repository.list_albums(
            user_id=user_id,
            viewer_id=viewer_id if viewer_id is not None else user_id,
            tab=tab,
            search=search,
        )

    async def execute_for_user(
        self,
        *,
        user_id: int,
        viewer_id: int | None = None,
        tab: str = "uploaded",
        search: str = "",
    ) -> tuple[VideoAlbumView, ...]:
        return await self.repository.list_albums(
            user_id=user_id,
            viewer_id=viewer_id,
            tab=tab,
            search=search,
        )


class CreateVideoAlbumUseCase:
    def __init__(self, repository: VideoRepository, transaction_manager: TransactionManager) -> None:
        self.repository = repository
        self.transaction_manager = transaction_manager

    async def execute(self, *, user: User, title: str) -> int:
        title = title.strip()
        if not title or len(title) > 80:
            raise ValueError("Название альбома должно содержать от 1 до 80 символов")
        async with self.transaction_manager:
            return await self.repository.create_album(user_id=user.require_id(), title=title)


class DeleteVideoUseCase:
    def __init__(self, repository: VideoRepository, media_storage: MediaStorage, transaction_manager: TransactionManager) -> None:
        self.repository, self.media_storage, self.transaction_manager = repository, media_storage, transaction_manager

    async def execute(self, *, user: User, video_id: int) -> bool:
        async with self.transaction_manager:
            media_id = await self.repository.delete_video(video_id=video_id, user_id=user.require_id())
            if media_id is None:
                return False
        await self.media_storage.delete(media_id)
        return True


class DeleteVideoAlbumUseCase:
    def __init__(self, repository: VideoRepository, media_storage: MediaStorage, transaction_manager: TransactionManager) -> None:
        self.repository, self.media_storage, self.transaction_manager = repository, media_storage, transaction_manager

    async def execute(self, *, user: User, album_id: int) -> bool:
        async with self.transaction_manager:
            media_ids = await self.repository.delete_album(album_id=album_id, user_id=user.require_id())
            if media_ids is None:
                return False
        for media_id in media_ids:
            await self.media_storage.delete(media_id)
        return True


class RecordVideoViewUseCase:
    def __init__(self, repository: VideoRepository, transaction_manager: TransactionManager) -> None:
        self.repository = repository
        self.transaction_manager = transaction_manager

    async def execute(self, *, video_id: int, user_id: int | None = None) -> int | None:
        async with self.transaction_manager:
            return await self.repository.record_view(video_id=video_id, user_id=user_id)


class ToggleVideoLikeUseCase:
    def __init__(self, repository: VideoRepository, transaction_manager: TransactionManager) -> None:
        self.repository = repository
        self.transaction_manager = transaction_manager

    async def execute(self, *, video_id: int, user: User) -> tuple[int, bool] | None:
        async with self.transaction_manager:
            return await self.repository.toggle_like(video_id=video_id, user_id=user.require_id())


class ToggleVideoBookmarkUseCase:
    def __init__(self, repository: VideoRepository, transaction_manager: TransactionManager) -> None:
        self.repository = repository
        self.transaction_manager = transaction_manager

    async def add(self, *, video_id: int, user: User) -> bool | None:
        async with self.transaction_manager:
            return await self.repository.add_bookmark(video_id=video_id, user_id=user.require_id())

    async def remove(self, *, video_id: int, user: User) -> bool | None:
        async with self.transaction_manager:
            return await self.repository.remove_bookmark(video_id=video_id, user_id=user.require_id())


class ToggleVideoFavoriteUseCase:
    def __init__(self, repository: VideoRepository, transaction_manager: TransactionManager) -> None:
        self.repository = repository
        self.transaction_manager = transaction_manager

    async def add(self, *, video_id: int, user: User) -> bool | None:
        async with self.transaction_manager:
            return await self.repository.add_favorite(video_id=video_id, user_id=user.require_id())

    async def remove(self, *, video_id: int, user: User) -> bool | None:
        async with self.transaction_manager:
            return await self.repository.remove_favorite(video_id=video_id, user_id=user.require_id())
