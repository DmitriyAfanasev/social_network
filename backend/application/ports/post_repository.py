from abc import ABC, abstractmethod
from collections.abc import Sequence

from backend.application.dto import RemovePostImageDTO
from backend.application.read_models import PostReadModel


class PostRepository(ABC):
    @abstractmethod
    async def create(
        self,
        content: str | None,
        author_id: int,
        image: str | None,
    ) -> PostReadModel:
        pass

    @abstractmethod
    async def get_by_id(self, post_id: int) -> PostReadModel | None:
        pass

    @abstractmethod
    async def update(
        self,
        post: PostReadModel,
        content: str | None,
        image: str | None,
        author_id: int,
    ) -> PostReadModel:
        pass

    @abstractmethod
    async def delete(self, post: PostReadModel) -> None:
        pass

    @abstractmethod
    async def remove_image(self, post: PostReadModel) -> RemovePostImageDTO:
        pass

    @abstractmethod
    async def get_paginated_by_likes(
        self,
        page: int = 1,
        limit: int = 10,
        current_user_id: int | None = None,
    ) -> tuple[Sequence[PostReadModel], int]:
        pass

    @abstractmethod
    async def get_all_by_author_id(
        self,
        author_id: int,
        current_user_id: int | None = None,
    ) -> Sequence[PostReadModel]:
        pass
