from abc import ABC, abstractmethod
from dataclasses import dataclass
from datetime import datetime


@dataclass(frozen=True, slots=True)
class SearchUserView:
    id: int
    username: str
    full_name: str
    avatar: str | None


@dataclass(frozen=True, slots=True)
class SearchPostView:
    id: int
    content: str | None
    author_id: int
    author_name: str
    created_at: datetime


@dataclass(frozen=True, slots=True)
class SearchVideoView:
    video_id: int
    media_id: int
    title: str
    owner_id: int | None
    owner_name: str
    created_at: datetime


@dataclass(frozen=True, slots=True)
class SearchResultView:
    users: tuple[SearchUserView, ...]
    posts: tuple[SearchPostView, ...]
    videos: tuple[SearchVideoView, ...]


class SearchRepository(ABC):
    @abstractmethod
    async def search(self, *, query: str, limit: int) -> SearchResultView: ...
