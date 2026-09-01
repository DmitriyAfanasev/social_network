from collections.abc import Sequence
from typing import cast

from sqlalchemy import and_, delete, exists, or_, select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload
from sqlalchemy.orm.attributes import InstrumentedAttribute
from sqlalchemy.sql.elements import ColumnElement
from sqlalchemy.sql.selectable import Exists

from backend.application.ports.friend_repository import FriendRepository as FriendPort
from backend.application.read_models import UserReadModel
from backend.infra.models.sqlalchemy import Friendship, Subscription, User


class FriendRepository(FriendPort):
    def __init__(self, session: AsyncSession) -> None:
        self.session = session

    async def user_exists(self, user_id: int) -> bool:
        statement = select(exists().where(User.id == user_id))
        return bool(await self.session.scalar(statement))

    async def list_friends(self, user_id: int) -> Sequence[UserReadModel]:
        statement = (
            select(User)
            .join(
                Friendship,
                or_(
                    Friendship.user_id == User.id,
                    Friendship.friend_id == User.id,
                ),
            )
            .where(
                or_(
                    Friendship.user_id == user_id,
                    Friendship.friend_id == user_id,
                ),
                User.id != user_id,
            )
            .options(selectinload(User.profile))
            .order_by(User.username)
        )
        result = await self.session.execute(statement)
        return cast(Sequence[UserReadModel], result.scalars().all())

    async def list_subscribers(self, user_id: int) -> Sequence[UserReadModel]:
        statement = (
            select(User)
            .join(Subscription, Subscription.subscriber_id == User.id)
            .where(
                Subscription.target_id == user_id,
                ~self._friend_exists_expression(user_id, User.id),
            )
            .options(selectinload(User.profile))
            .order_by(User.username)
        )
        result = await self.session.execute(statement)
        return cast(Sequence[UserReadModel], result.scalars().all())

    async def list_subscriptions(self, user_id: int) -> Sequence[UserReadModel]:
        statement = (
            select(User)
            .join(Subscription, Subscription.target_id == User.id)
            .where(
                Subscription.subscriber_id == user_id,
                ~self._friend_exists_expression(user_id, User.id),
            )
            .options(selectinload(User.profile))
            .order_by(User.username)
        )
        result = await self.session.execute(statement)
        return cast(Sequence[UserReadModel], result.scalars().all())

    async def is_friend(self, user_id: int, friend_id: int) -> bool:
        left_id, right_id = self._normalize_pair(user_id, friend_id)
        statement = select(
            exists().where(
                Friendship.user_id == left_id,
                Friendship.friend_id == right_id,
            )
        )
        return bool(await self.session.scalar(statement))

    async def are_friends_of_friends(self, user_id: int, target_id: int) -> bool:
        current_friends = {user.id for user in await self.list_friends(user_id)}
        target_friends = {user.id for user in await self.list_friends(target_id)}
        return bool(current_friends & target_friends)

    async def is_subscribed(self, subscriber_id: int, target_id: int) -> bool:
        statement = select(
            exists().where(
                Subscription.subscriber_id == subscriber_id,
                Subscription.target_id == target_id,
            )
        )
        return bool(await self.session.scalar(statement))

    async def subscribe(self, subscriber_id: int, target_id: int) -> None:
        if await self.is_subscribed(subscriber_id, target_id):
            return

        self.session.add(Subscription(subscriber_id=subscriber_id, target_id=target_id))
        await self.session.flush()

    async def create_friendship(self, user_id: int, friend_id: int) -> None:
        left_id, right_id = self._normalize_pair(user_id, friend_id)
        if await self.is_friend(left_id, right_id):
            return

        self.session.add(Friendship(user_id=left_id, friend_id=right_id))
        await self.session.flush()

    async def remove_friend(self, user_id: int, friend_id: int) -> bool:
        left_id, right_id = self._normalize_pair(user_id, friend_id)
        if not await self.is_friend(left_id, right_id):
            return False

        await self.session.execute(
            delete(Friendship).where(
                Friendship.user_id == left_id,
                Friendship.friend_id == right_id,
            )
        )
        return True

    async def unsubscribe(self, subscriber_id: int, target_id: int) -> bool:
        if not await self.is_subscribed(subscriber_id, target_id):
            return False

        await self.session.execute(
            delete(Subscription).where(
                Subscription.subscriber_id == subscriber_id,
                Subscription.target_id == target_id,
            )
        )
        return True

    @staticmethod
    def _normalize_pair(user_id: int, friend_id: int) -> tuple[int, int]:
        return (user_id, friend_id) if user_id < friend_id else (friend_id, user_id)

    @staticmethod
    def _friend_exists_expression(
        user_id: int,
        other_user_id: int | ColumnElement[int] | InstrumentedAttribute[int],
    ) -> Exists:
        return exists().where(
            or_(
                and_(
                    Friendship.user_id == user_id,
                    Friendship.friend_id == other_user_id,
                ),
                and_(
                    Friendship.user_id == other_user_id,
                    Friendship.friend_id == user_id,
                ),
            )
        )
