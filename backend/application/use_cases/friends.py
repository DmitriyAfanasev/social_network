from backend.application.analytics_events import build_analytics_event_payload
from backend.application.event_types import (
    FRIEND_ACCEPTED_EVENT,
    FRIEND_REMOVED_EVENT,
    FRIEND_REQUEST_CANCELLED_EVENT,
    FRIEND_REQUESTED_EVENT,
)
from backend.application.events import IntegrationEvent
from backend.application.exceptions import NotFoundError, ValidationAppError
from backend.application.ports.block_repository import BlockRepository
from backend.application.ports.friend_repository import FriendRepository
from backend.application.ports.outbox_repository import OutboxRepository
from backend.application.ports.profile_repository import ProfileRepository
from backend.application.ports.transaction_manager import TransactionManager
from backend.application.results import (
    FriendActionResult,
    FriendRecommendation,
    FriendRecommendationsResult,
    FriendsResult,
)
from backend.domain.user.entity import User
from backend.domain.user.policy import InteractionPolicy, RelationshipFacts


class GetFriendsUseCase:
    def __init__(self, friend_repository: FriendRepository) -> None:
        self.friend_repository = friend_repository

    async def execute(self, current_user: User) -> FriendsResult:
        current_user_id = current_user.require_id()
        friends = await self.friend_repository.list_friends(current_user_id)
        subscribers = await self.friend_repository.list_subscribers(current_user_id)
        subscriptions = await self.friend_repository.list_subscriptions(current_user_id)
        return FriendsResult(
            current_user=current_user,
            friends=friends,
            subscribers=subscribers,
            subscriptions=subscriptions,
        )


class GetFriendRecommendationsUseCase:
    def __init__(self, friend_repository: FriendRepository, block_repository: BlockRepository) -> None:
        self.friend_repository = friend_repository
        self.block_repository = block_repository

    async def execute(self, current_user: User, limit: int = 10) -> FriendRecommendationsResult:
        if not 1 <= limit <= 50:
            raise ValidationAppError("Параметр limit должен быть от 1 до 50")
        user_id = current_user.require_id()
        direct_friends = await self.friend_repository.list_friends(user_id)
        direct_ids = {friend.id for friend in direct_friends}
        candidates: dict[int, int] = {}
        users_by_id = {}
        for friend in direct_friends:
            for candidate in await self.friend_repository.list_friends(friend.id):
                if candidate.id == user_id or candidate.id in direct_ids:
                    continue
                if await self.block_repository.is_blocked(user_id, candidate.id):
                    continue
                if await self.friend_repository.is_subscribed(user_id, candidate.id):
                    continue
                candidates[candidate.id] = candidates.get(candidate.id, 0) + 1
                users_by_id[candidate.id] = candidate

        recommendations = [
            FriendRecommendation(user=users_by_id[candidate_id], common_friends=count)
            for candidate_id, count in sorted(
                candidates.items(), key=lambda item: (-item[1], item[0])
            )[:limit]
        ]
        return FriendRecommendationsResult(current_user=current_user, recommendations=recommendations)


class AddFriendUseCase:
    def __init__(
        self,
        friend_repository: FriendRepository,
        outbox_repository: OutboxRepository,
        transaction_manager: TransactionManager,
        block_repository: BlockRepository,
        profile_repository: ProfileRepository,
    ) -> None:
        self.friend_repository = friend_repository
        self.profile_repository = profile_repository
        self.block_repository = block_repository
        self.outbox_repository = outbox_repository
        self.transaction_manager = transaction_manager

    async def execute(self, current_user: User, friend_id: int) -> FriendActionResult:
        current_user_id = current_user.require_id()
        self._validate_pair(current_user_id, friend_id)
        if not await self.friend_repository.user_exists(friend_id):
            raise NotFoundError("User not found")
        is_blocked = await self.block_repository.is_blocked(current_user_id, friend_id)
        target = await self.profile_repository.get_by_id(friend_id)
        is_friend = await self.friend_repository.is_friend(current_user_id, friend_id)
        are_friends_of_friends = (
            False
            if is_friend
            else await self.friend_repository.are_friends_of_friends(current_user_id, friend_id)
        )
        facts = RelationshipFacts(
            is_self=current_user_id == friend_id,
            is_blocked=is_blocked,
            is_friend=is_friend,
            are_friends_of_friends=are_friends_of_friends,
        )
        friend_request_policy = target.profile.friend_request_policy if target.profile else "everyone"
        if not InteractionPolicy.can_send_friend_request(friend_request_policy, facts):
            raise ValidationAppError("Пользователь запретил входящие заявки в друзья")

        async with self.transaction_manager:
            await self.friend_repository.subscribe(current_user_id, friend_id)
            is_friend = await self.friend_repository.is_subscribed(friend_id, current_user_id)
            if is_friend:
                await self.friend_repository.create_friendship(current_user_id, friend_id)
            await self.outbox_repository.add(
                IntegrationEvent(
                    event_type=FRIEND_ACCEPTED_EVENT if is_friend else FRIEND_REQUESTED_EVENT,
                    payload=build_analytics_event_payload(
                        user_id=current_user_id,
                        entity_type="friendship",
                        entity_id=friend_id,
                        data={"target_user_id": friend_id, "action": "accepted" if is_friend else "requested"},
                    ),
                )
            )

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
        outbox_repository: OutboxRepository,
        transaction_manager: TransactionManager,
    ) -> None:
        self.friend_repository = friend_repository
        self.outbox_repository = outbox_repository
        self.transaction_manager = transaction_manager

    async def execute(self, current_user: User, friend_id: int) -> FriendActionResult:
        current_user_id = current_user.require_id()
        if current_user_id == friend_id:
            raise ValidationAppError("Нельзя удалить себя из друзей")

        async with self.transaction_manager:
            removed = await self.friend_repository.remove_friend(current_user_id, friend_id)
            if removed:
                await self.friend_repository.unsubscribe(current_user_id, friend_id)
                await self.outbox_repository.add(
                    IntegrationEvent(
                        event_type=FRIEND_REMOVED_EVENT,
                        payload=build_analytics_event_payload(
                            user_id=current_user_id,
                            entity_type="friendship",
                            entity_id=friend_id,
                            data={"target_user_id": friend_id, "action": "removed"},
                        ),
                    )
                )

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
        outbox_repository: OutboxRepository,
        transaction_manager: TransactionManager,
    ) -> None:
        self.friend_repository = friend_repository
        self.outbox_repository = outbox_repository
        self.transaction_manager = transaction_manager

    async def execute(self, current_user: User, target_id: int) -> FriendActionResult:
        current_user_id = current_user.require_id()
        if current_user_id == target_id:
            raise ValidationAppError("Нельзя отписаться от себя")

        async with self.transaction_manager:
            removed = await self.friend_repository.unsubscribe(current_user_id, target_id)
            if removed:
                await self.outbox_repository.add(
                    IntegrationEvent(
                        event_type=FRIEND_REQUEST_CANCELLED_EVENT,
                        payload=build_analytics_event_payload(
                            user_id=current_user_id,
                            entity_type="friendship",
                            entity_id=target_id,
                            data={"target_user_id": target_id, "action": "cancelled"},
                        ),
                    )
                )

        return FriendActionResult(
            success=removed,
            is_friend=await self.friend_repository.is_friend(current_user_id, target_id),
            is_subscribed=False,
            is_subscribed_to_current=await self.friend_repository.is_subscribed(target_id, current_user_id),
            message="Подписка отменена" if removed else "Подписка не найдена",
        )
