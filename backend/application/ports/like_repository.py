from abc import ABC, abstractmethod

from backend.application.dto import ToggleLikeDTO


class LikeRepository(ABC):
    @abstractmethod
    async def toggle_post_like(self, user_id: int, post_id: int) -> ToggleLikeDTO:
        pass

    @abstractmethod
    async def toggle_comment_like(self, user_id: int, comment_id: int) -> ToggleLikeDTO:
        pass
