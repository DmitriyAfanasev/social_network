from typing import Any
from unittest.mock import AsyncMock

import pytest

from backend.application.exceptions import PermissionDeniedError, ValidationAppError
from backend.application.use_cases.blocks import BlockUserUseCase
from backend.application.use_cases.friends import AddFriendUseCase
from backend.application.use_cases.messages import MessagingUseCase
from backend.application.use_cases.profiles import GetProfileUseCase
from backend.domain.user.entity import User
from backend.domain.user.entity.profile import Profile
from tests.fakes import FakeTransaction


# вынести их в tests/factories/domain.py; фикстуры оставить для зависимостей и ресурсов.
# TODO: покрыть RBAC и проверить, что каждое административное действие
# действительно защищено application permission-проверкой.


def user(user_id: int) -> User:
    return User(
        id=user_id,
        username=f"user{user_id}",
        email=f"user{user_id}@example.com",
        hashed_password="hash",
    )

# TODO: если make_user/target_user начнут повторяться в нескольких файлах,
def target_user(user_id: int, **profile_fields: Any) -> User:
    return User(
        id=user_id,
        username=f"user{user_id}",
        email=f"user{user_id}@example.com",
        hashed_password="hash",
        profile=Profile(**profile_fields),
    )


@pytest.mark.application
@pytest.mark.asyncio
async def test_friend_request_is_rejected_when_target_blocks_actor() -> None:
    friends = AsyncMock()
    friends.user_exists.return_value = True
    friends.is_friend.return_value = False
    blocks = AsyncMock()
    blocks.is_blocked.return_value = True
    profiles = AsyncMock()
    profiles.get_by_id.return_value = user(2)

    with pytest.raises(ValidationAppError, match="запретил"):
        await AddFriendUseCase(
            friends, AsyncMock(), FakeTransaction(), blocks, profiles
        ).execute(user(1), 2)

    friends.subscribe.assert_not_awaited()


@pytest.mark.application
@pytest.mark.asyncio
async def test_friend_request_respects_friends_of_friends_policy() -> None:
    friends = AsyncMock()
    friends.user_exists.return_value = True
    friends.is_friend.return_value = False
    friends.are_friends_of_friends.return_value = True
    blocks = AsyncMock()
    blocks.is_blocked.return_value = False
    profiles = AsyncMock()
    profiles.get_by_id.return_value = User(
        id=2,
        username="target",
        email="target@example.com",
        hashed_password="hash",
        profile=Profile(friend_request_policy="friends_of_friends"),
    )
    friends.is_subscribed.return_value = False

    await AddFriendUseCase(
        friends, AsyncMock(), FakeTransaction(), blocks, profiles
    ).execute(user(1), 2)

    friends.subscribe.assert_awaited_once_with(1, 2)


@pytest.mark.application
@pytest.mark.asyncio
async def test_block_removes_friendship_and_both_subscriptions() -> None:
    blocks = AsyncMock()
    blocks.user_exists.return_value = True
    friends = AsyncMock()

    result = await BlockUserUseCase(
        blocks, friends, FakeTransaction()
    ).execute(user(1), 2)

    assert result.message == "Пользователь заблокирован"
    blocks.block.assert_awaited_once_with(1, 2)
    friends.remove_friend.assert_awaited_once_with(1, 2)
    friends.unsubscribe.assert_any_await(1, 2)
    friends.unsubscribe.assert_any_await(2, 1)


@pytest.mark.application
@pytest.mark.asyncio
async def test_message_policy_rejects_blocked_target_before_writing() -> None:
    repository = AsyncMock()
    blocks = AsyncMock()
    blocks.is_blocked.return_value = True
    profiles = AsyncMock()
    profiles.get_by_id.return_value = user(2)
    friends = AsyncMock()

    use_case = MessagingUseCase(
        repository,
        AsyncMock(),
        FakeTransaction(),
        profile_repository=profiles,
        friend_repository=friends,
        block_repository=blocks,
    )

    with pytest.raises(PermissionDeniedError):
        await use_case._ensure_can_message(1, 2)

    repository.create_message.assert_not_awaited()


@pytest.mark.application
@pytest.mark.asyncio
async def test_profile_is_hidden_when_target_blocks_actor() -> None:
    profiles = AsyncMock()
    profiles.get_by_id.return_value = target_user(2)
    friends = AsyncMock()
    blocks = AsyncMock()
    blocks.is_blocked.return_value = True

    use_case = GetProfileUseCase(profiles, AsyncMock(), friends, blocks)

    with pytest.raises(PermissionDeniedError, match="недоступен"):
        await use_case.execute(user(1), 2)

    friends.is_friend.assert_not_awaited()


@pytest.mark.application
@pytest.mark.asyncio
@pytest.mark.parametrize(
    ("visibility", "is_friend", "friends_of_friends", "visible"),
    [
        ("everyone", False, False, True),
        ("friends", True, False, True),
        ("friends", False, True, False),
        ("friends_of_friends", False, True, True),
        ("nobody", True, True, False),
    ],
)
async def test_profile_visibility_uses_relationship_facts(
    visibility: str,
    is_friend: bool,
    friends_of_friends: bool,
    visible: bool,
) -> None:
    profiles = AsyncMock()
    profiles.get_by_id.return_value = target_user(2, profile_visibility=visibility)
    friends = AsyncMock()
    friends.is_friend.return_value = is_friend
    friends.are_friends_of_friends.return_value = friends_of_friends
    friends.is_subscribed.return_value = False
    blocks = AsyncMock()
    blocks.is_blocked.return_value = False

    use_case = GetProfileUseCase(profiles, AsyncMock(), friends, blocks)

    if visible:
        result = await use_case.execute(user(1), 2)
        assert result.user.id == 2
    else:
        with pytest.raises(PermissionDeniedError, match="ограничил"):
            await use_case.execute(user(1), 2)


@pytest.mark.application
@pytest.mark.asyncio
async def test_profile_exposes_message_permission_from_target_policy() -> None:
    profiles = AsyncMock()
    profiles.get_by_id.return_value = target_user(
        2,
        message_policy="friends",
        friend_request_policy="friends",
    )
    friends = AsyncMock()
    friends.is_friend.return_value = False
    friends.are_friends_of_friends.return_value = False
    friends.is_subscribed.return_value = False
    blocks = AsyncMock()
    blocks.is_blocked.return_value = False
    posts = AsyncMock()
    posts.get_all_by_author_id.return_value = []

    result = await GetProfileUseCase(profiles, posts, friends, blocks).execute(user(1), 2)

    assert result.can_send_message is False
    assert result.can_send_friend_request is False
