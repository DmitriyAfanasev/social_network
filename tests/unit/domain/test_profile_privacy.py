import pytest

from backend.domain.user.entity.profile import can_access_profile


pytestmark = [pytest.mark.domain]


@pytest.mark.parametrize(
    ("policy", "is_friend", "friends_of_friends", "expected"),
    [
        ("everyone", False, False, True),
        ("friends", True, False, True),
        ("friends", False, True, False),
        ("friends_of_friends", False, True, True),
        ("friends_of_friends", False, False, False),
        ("nobody", True, True, False),
    ],
)
def test_profile_visibility_policy(
    policy: str,
    is_friend: bool,
    friends_of_friends: bool,
    expected: bool,
) -> None:
    assert can_access_profile(
        policy,
        is_friend=is_friend,
        are_friends_of_friends=friends_of_friends,
    ) is expected
