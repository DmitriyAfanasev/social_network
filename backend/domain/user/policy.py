from dataclasses import dataclass

from backend.domain.user.entity.profile import can_access_profile


@dataclass(frozen=True, slots=True)
class RelationshipFacts:
    """Facts about the actor and target used by interaction policies.

    The policy deliberately knows nothing about persistence or transport. Facts
    are assembled by an application use case from repository ports.
    """

    is_self: bool = False
    is_blocked: bool = False
    is_friend: bool = False
    are_friends_of_friends: bool = False


class InteractionPolicy:
    """Domain decisions for interactions between two users."""

    @staticmethod
    def can_view_profile(policy: str, facts: RelationshipFacts) -> bool:
        if facts.is_self:
            return True
        if facts.is_blocked:
            return False
        return can_access_profile(
            policy,
            is_friend=facts.is_friend,
            are_friends_of_friends=facts.are_friends_of_friends,
        )

    @staticmethod
    def can_send_message(policy: str, facts: RelationshipFacts) -> bool:
        if facts.is_blocked:
            return False
        if facts.is_self:
            return True
        return InteractionPolicy.can_access_target(policy, facts)

    @staticmethod
    def can_send_friend_request(policy: str, facts: RelationshipFacts) -> bool:
        if facts.is_self or facts.is_blocked:
            return False
        return InteractionPolicy.can_access_target(policy, facts)

    @staticmethod
    def can_access_target(policy: str, facts: RelationshipFacts) -> bool:
        return can_access_profile(
            policy,
            is_friend=facts.is_friend,
            are_friends_of_friends=facts.are_friends_of_friends,
        )
