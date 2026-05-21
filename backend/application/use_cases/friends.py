from typing import cast

from backend.application.exceptions import NotFoundError, ValidationAppError
from backend.application.ports.friend_repository import FriendRepository
from backend.application.ports.transaction_manager import TransactionManager
from backend.application.results import FriendActionResult, FriendsResult
from backend.domain.user.entity import User


class GetFriendsUseCase:
    def __init__(self, friend_repository: FriendRepository) -> None:
        self.friend_repository = friend_repository

    async def execute(self, current_user: User) -> FriendsResult:
        current_user_id = cast(int, current_user.id)
        friends = await self.friend_repository.list_friends(current_user_id)
        subscribers = await self.friend_repository.list_subscribers(current_user_id)
        subscriptions = await self.friend_repository.list_subscriptions(current_user_id)
        return FriendsResult(
            current_user=current_user,
            friends=friends,
            subscribers=subscribers,
            subscriptions=subscriptions,
        )


class AddFriendUseCase:
    def __init__(
        self,
        friend_repository: FriendRepository,
        transaction_manager: TransactionManager,
    ) -> None:
        self.friend_repository = friend_repository
        self.transaction_manager = transaction_manager

    async def execute(self, current_user: User, friend_id: int) -> FriendActionResult:
        current_user_id = cast(int, current_user.id)
        self._validate_pair(current_user_id, friend_id)
        if not await self.friend_repository.user_exists(friend_id):
            raise NotFoundError("User not found")

        async with self.transaction_manager:
            await self.friend_repository.subscribe(current_user_id, friend_id)
            is_friend = await self.friend_repository.is_subscribed(friend_id, current_user_id)
            if is_friend:
                await self.friend_repository.create_friendship(current_user_id, friend_id)

        return FriendActionResult(
            success=True,
            is_friend=is_friend,
            is_subscribed=True,
            is_subscribed_to_current=is_friend,
            message="Вы теперь друзья" if is_friend else "Заявка в друзья отправлена",
        )

    @staticmethod
    def _validate_pair(current_user_id: int, friend_id: int) -> None:
        if current_user_id == friend_id:
            raise ValidationAppError("Нельзя добавить себя в друзья")


class RemoveFriendUseCase:
    def __init__(
        self,
        friend_repository: FriendRepository,
        transaction_manager: TransactionManager,
    ) -> None:
        self.friend_repository = friend_repository
        self.transaction_manager = transaction_manager

    async def execute(self, current_user: User, friend_id: int) -> FriendActionResult:
        current_user_id = cast(int, current_user.id)
        if current_user_id == friend_id:
            raise ValidationAppError("Нельзя удалить себя из друзей")

        async with self.transaction_manager:
            removed = await self.friend_repository.remove_friend(current_user_id, friend_id)
            if removed:
                await self.friend_repository.unsubscribe(current_user_id, friend_id)

        return FriendActionResult(
            success=removed,
            is_friend=False,
            is_subscribed=False,
            is_subscribed_to_current=await self.friend_repository.is_subscribed(friend_id, current_user_id),
            message="Пользователь удалён из друзей" if removed else "Пользователь не был в друзьях",
        )


class CancelSubscriptionUseCase:
    def __init__(
        self,
        friend_repository: FriendRepository,
        transaction_manager: TransactionManager,
    ) -> None:
        self.friend_repository = friend_repository
        self.transaction_manager = transaction_manager

    async def execute(self, current_user: User, target_id: int) -> FriendActionResult:
        current_user_id = cast(int, current_user.id)
        if current_user_id == target_id:
            raise ValidationAppError("Нельзя отписаться от себя")

        async with self.transaction_manager:
            removed = await self.friend_repository.unsubscribe(current_user_id, target_id)

        return FriendActionResult(
            success=removed,
            is_friend=await self.friend_repository.is_friend(current_user_id, target_id),
            is_subscribed=False,
            is_subscribed_to_current=await self.friend_repository.is_subscribed(target_id, current_user_id),
            message="Подписка отменена" if removed else "Подписка не найдена",
        )
