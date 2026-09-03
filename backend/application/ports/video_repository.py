from abc import ABC, abstractmethod
from dataclasses import dataclass
from datetime import datetime


@dataclass(frozen=True, slots=True)
class VideoRenditionView:
    height: int
    object_key: str
    content_type: str
    size: int


@dataclass(frozen=True, slots=True)
class VideoView:
    id: int
    media_id: int
    status: str
    error_message: str | None
    renditions: tuple[VideoRenditionView, ...]
    original_filename: str = ""
    created_at: datetime | None = None
    duration: float | None = None
    title: str = "Видео"
    owner_id: int | None = None
    owner_name: str = "пользователя"
    views_count: int = 0
    likes_count: int = 0
    is_liked_by_current: bool = False
    is_bookmarked_by_current: bool = False
    is_favorited_by_current: bool = False


@dataclass(frozen=True, slots=True)
class VideoRenditionInput:
    height: int
    object_key: str
    content_type: str = "video/mp4"
    size: int = 0
    width: int | None = None
    duration: float | None = None


@dataclass(frozen=True, slots=True)
class VideoAlbumView:
    id: int
    title: str
    videos: tuple[VideoView, ...]


class VideoRepository(ABC):
    @abstractmethod
    async def get_owner_name(self, *, user_id: int) -> str | None: ...

    @abstractmethod
    async def list_albums(
        self,
        *,
        user_id: int,
        viewer_id: int | None = None,
        tab: str = "uploaded",
        search: str = "",
    ) -> tuple[VideoAlbumView, ...]: ...

    @abstractmethod
    async def create_album(self, *, user_id: int, title: str) -> int: ...

    @abstractmethod
    async def album_belongs_to_user(self, *, album_id: int, user_id: int) -> bool: ...

    @abstractmethod
    async def delete_video(self, *, video_id: int, user_id: int) -> int | None: ...

    @abstractmethod
    async def delete_album(self, *, album_id: int, user_id: int) -> tuple[int, ...] | None: ...

    @abstractmethod
    async def source_object_key(self, *, media_id: int) -> str: ...

    @abstractmethod
    async def create(self, *, media_id: int, album_id: int | None = None, title: str = "Видео") -> int: ...

    @abstractmethod
    async def get_for_user(self, *, video_id: int, user_id: int | None = None) -> VideoView | None: ...

    @abstractmethod
    async def record_view(self, *, video_id: int, user_id: int | None = None) -> int | None: ...

    @abstractmethod
    async def toggle_like(self, *, video_id: int, user_id: int) -> tuple[int, bool] | None: ...

    @abstractmethod
    async def add_bookmark(self, *, video_id: int, user_id: int) -> bool | None: ...

    @abstractmethod
    async def remove_bookmark(self, *, video_id: int, user_id: int) -> bool | None: ...

    @abstractmethod
    async def add_favorite(self, *, video_id: int, user_id: int) -> bool | None: ...

    @abstractmethod
    async def remove_favorite(self, *, video_id: int, user_id: int) -> bool | None: ...

    @abstractmethod
    async def complete_transcoding(
        self,
        *,
        video_id: int,
        status: str,
        renditions: tuple[VideoRenditionInput, ...] = (),
        duration: float | None = None,
        error_message: str | None = None,
    ) -> bool: ...
