from dataclasses import dataclass
from datetime import date


PROFILE_VISIBILITY_POLICIES = ("everyone", "friends", "friends_of_friends", "nobody")


def can_access_profile(policy: str, *, is_friend: bool, are_friends_of_friends: bool) -> bool:
    return policy == "everyone" or (
        policy == "friends" and is_friend
    ) or (
        policy == "friends_of_friends" and (is_friend or are_friends_of_friends)
    )


@dataclass(kw_only=True)
class Profile:
    first_name: str | None = None
    last_name: str | None = None
    middle_name: str | None = None
    birth_date: date | None = None
    gender: str | None = None
    phone_number: str | None = None
    country: str | None = None
    city: str | None = None
    street: str | None = None
    bio: str | None = None
    status: str | None = None
    avatar: str | None = None
    profile_visibility: str = "everyone"
    friend_request_policy: str = "everyone"
    message_policy: str = "everyone"
    show_email: bool = False
    show_phone: bool = False
    show_birth_date: bool = True
    show_friends: bool = True
    show_posts: bool = True

    @property
    def full_name(self) -> str | None:
        if self.first_name and self.last_name:
            return f"{self.first_name} {self.last_name}"
        return None
