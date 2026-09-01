from abc import ABC, abstractmethod

from backend.application.read_models import CommentReadModel, PostReadModel


class CommentRepository(ABC):
    @abstractmethod
    async def get_post_by_id(self, post_id: int) -> PostReadModel | None:
        pass

    @abstractmethod
    async def get_comment_by_id(self, comment_id: int) -> CommentReadModel | None:
        pass

    @abstractmethod
    async def get_comment_depth(self, comment_id: int) -> int:
        pass

    @abstractmethod
    async def create(
        self,
        user_id: int,
        post_id: int,
        text: str,
        parent_id: int | None = None,
    ) -> CommentReadModel:
        pass

    @abstractmethod
    async def update(self, comment: CommentReadModel, text: str) -> CommentReadModel:
        pass

    @abstractmethod
    async def delete(self, comment: CommentReadModel) -> None:
        pass

    @abstractmethod
    async def get_paginated(
        self,
        post_id: int,
        offset: int,
        limit: int,
    ) -> tuple[list[CommentReadModel], bool]:
        pass
