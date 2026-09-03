from abc import ABC, abstractmethod
from dataclasses import dataclass
from datetime import datetime


@dataclass(frozen=True, slots=True)
class MusicTrackView:
    id: int
    media_id: int
    user_id: int
    title: str
    artist: str
    duration: float | None
    created_at: datetime


class MusicRepository(ABC):
    @abstractmethod
    async def list_by_user(self, *, user_id: int) -> tuple[MusicTrackView, ...]: ...

    @abstractmethod
    async def create(
        self,
        *,
        user_id: int,
        media_id: int,
        title: str,
        artist: str,
    ) -> MusicTrackView: ...

    @abstractmethod
    async def delete(self, *, track_id: int, user_id: int) -> int | None: ...
