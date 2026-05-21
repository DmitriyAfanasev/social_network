from abc import ABC, abstractmethod
from collections.abc import Sequence

from backend.application.read_models import UserReadModel


class FriendRepository(ABC):
    @abstractmethod
    async def user_exists(self, user_id: int) -> bool:
        pass

    @abstractmethod
    async def list_friends(self, user_id: int) -> Sequence[UserReadModel]:
        pass

    @abstractmethod
    async def list_subscribers(self, user_id: int) -> Sequence[UserReadModel]:
        pass

    @abstractmethod
    async def list_subscriptions(self, user_id: int) -> Sequence[UserReadModel]:
        pass

    @abstractmethod
    async def is_friend(self, user_id: int, friend_id: int) -> bool:
        pass

    @abstractmethod
    async def is_subscribed(self, subscriber_id: int, target_id: int) -> bool:
        pass

    @abstractmethod
    async def subscribe(self, subscriber_id: int, target_id: int) -> None:
        pass

    @abstractmethod
    async def create_friendship(self, user_id: int, friend_id: int) -> None:
        pass

    @abstractmethod
    async def remove_friend(self, user_id: int, friend_id: int) -> bool:
        pass

    @abstractmethod
    async def unsubscribe(self, subscriber_id: int, target_id: int) -> bool:
        pass
