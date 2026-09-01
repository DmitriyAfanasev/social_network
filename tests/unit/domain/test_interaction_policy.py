import pytest

from backend.domain.user.policy import InteractionPolicy, RelationshipFacts


@pytest.mark.domain
@pytest.mark.parametrize(
    ("policy", "facts", "expected"),
    [
        ("everyone", RelationshipFacts(), True),
        ("friends", RelationshipFacts(is_friend=True), True),
        ("friends", RelationshipFacts(are_friends_of_friends=True), False),
        ("friends_of_friends", RelationshipFacts(are_friends_of_friends=True), True),
        ("nobody", RelationshipFacts(is_friend=True), False),
        ("everyone", RelationshipFacts(is_blocked=True), False),
        ("nobody", RelationshipFacts(is_self=True), True),
    ],
)
def test_profile_visibility_is_decided_from_relationship_facts(
    policy: str,
    facts: RelationshipFacts,
    expected: bool,
) -> None:
    assert InteractionPolicy.can_view_profile(policy, facts) is expected


@pytest.mark.domain
@pytest.mark.parametrize(
    ("policy", "facts", "expected"),
    [
        ("everyone", RelationshipFacts(), True),
        ("friends", RelationshipFacts(is_friend=True), True),
        ("friends_of_friends", RelationshipFacts(are_friends_of_friends=True), True),
        ("friends_of_friends", RelationshipFacts(), False),
        ("everyone", RelationshipFacts(is_blocked=True), False),
        ("everyone", RelationshipFacts(is_self=True), True),
    ],
)
def test_message_permission_respects_block_and_relationship_policy(
    policy: str,
    facts: RelationshipFacts,
    expected: bool,
) -> None:
    assert InteractionPolicy.can_send_message(policy, facts) is expected


@pytest.mark.domain
def test_friend_request_cannot_be_sent_to_self_or_blocked_user() -> None:
    assert not InteractionPolicy.can_send_friend_request(
        "everyone", RelationshipFacts(is_self=True)
    )
    assert not InteractionPolicy.can_send_friend_request(
        "everyone", RelationshipFacts(is_blocked=True)
    )


@pytest.mark.domain
def test_friend_request_uses_target_policy() -> None:
    assert InteractionPolicy.can_send_friend_request(
        "friends_of_friends", RelationshipFacts(are_friends_of_friends=True)
    )
    assert not InteractionPolicy.can_send_friend_request(
        "friends_of_friends", RelationshipFacts()
    )
