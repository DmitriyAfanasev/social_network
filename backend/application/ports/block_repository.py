from abc import ABC, abstractmethod


class BlockRepository(ABC):
    @abstractmethod
    async def user_exists(self, user_id: int) -> bool:
        pass

    @abstractmethod
    async def is_blocked(self, user_id: int, other_user_id: int) -> bool:
        """Return true when either user blocks the other."""
        pass

    @abstractmethod
    async def block(self, blocker_id: int, blocked_id: int) -> None:
        pass

    @abstractmethod
    async def unblock(self, blocker_id: int, blocked_id: int) -> bool:
        pass

