from dataclasses import dataclass


@dataclass(frozen=True)
class AuthTokensDTO:
    access_token: str
    refresh_token: str


@dataclass(frozen=True)
class RemovePostImageDTO:
    success: bool
    message: str


@dataclass(frozen=True)
class ToggleLikeDTO:
    success: bool
    action: str
    likes_count: int
    liked: bool
