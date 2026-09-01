from unittest.mock import AsyncMock

import pytest

from backend.application.exceptions import ValidationAppError
from backend.application.use_cases.friends import GetFriendRecommendationsUseCase
from backend.domain.user.entity import User


def make_user(user_id: int) -> User:
    return User(id=user_id, username=f"u{user_id}", email=f"u{user_id}@e.com", hashed_password="hash")


@pytest.mark.application
@pytest.mark.asyncio
async def test_recommendations_rank_by_common_friends_and_exclude_blocked_or_subscribed() -> None:
    friends = AsyncMock()
    blocks = AsyncMock()
    current = make_user(1)
    common_a = make_user(2)
    common_b = make_user(3)
    candidate = make_user(4)
    excluded = make_user(5)
    friends.list_friends.side_effect = [[common_a, common_b], [candidate], [candidate, excluded]]
    blocks.is_blocked.side_effect = lambda _user_id, candidate_id: _user_id == 1 and candidate_id == 5
    friends.is_subscribed.side_effect = lambda _user_id, candidate_id: candidate_id == 5

    result = await GetFriendRecommendationsUseCase(friends, blocks).execute(current)

    assert [(item.user.id, item.common_friends) for item in result.recommendations] == [(4, 2)]


@pytest.mark.application
@pytest.mark.asyncio
async def test_recommendations_validate_limit_before_repository_access() -> None:
    friends = AsyncMock()
    with pytest.raises(ValidationAppError):
        await GetFriendRecommendationsUseCase(friends, AsyncMock()).execute(make_user(1), limit=0)
    friends.list_friends.assert_not_awaited()
