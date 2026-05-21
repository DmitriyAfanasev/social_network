from collections.abc import Sequence
from dataclasses import dataclass
from datetime import datetime

from backend.application.dto import AuthTokensDTO
from backend.application.read_models import (
    CommentReadModel,
    PostReadModel,
    ProfileAvatarReadModel,
    UserReadModel,
)
from backend.domain.user.entity import User


@dataclass(frozen=True)
class MessageResult:
    message: str


@dataclass(frozen=True)
class AccessTokenResult:
    access_token: str


@dataclass(frozen=True)
class AuthResult:
    user: User
    tokens: AuthTokensDTO


@dataclass(frozen=True)
class FeedResult:
    current_user: User | None
    posts: Sequence[PostReadModel]
    page: int
    total_pages: int


@dataclass(frozen=True)
class PostResult:
    post: PostReadModel
    author: User | None = None


@dataclass(frozen=True)
class RemovePostImageResult:
    success: bool
    message: str


@dataclass(frozen=True)
class CommentResult:
    comment: CommentReadModel
    author: User | None = None


@dataclass(frozen=True)
class CommentsPageResult:
    comments: Sequence[CommentReadModel]
    has_more: bool
    offset: int
    limit: int
    post_id: int


@dataclass(frozen=True)
class ToggleLikeResult:
    success: bool
    action: str
    likes_count: int
    liked: bool


@dataclass(frozen=True)
class ProfileResult:
    user: User
    is_own_profile: bool
    is_friend: bool
    is_subscribed: bool
    is_subscribed_to_current: bool
    current_user: User
    posts: Sequence[PostReadModel]


@dataclass(frozen=True)
class UserResult:
    user: User


@dataclass(frozen=True)
class AvatarUploadResult:
    message: str
    avatar_url: str


@dataclass(frozen=True)
class AvatarRemoveResult:
    new_avatar: str
    message: str


@dataclass(frozen=True)
class AvatarHistoryResult:
    current_avatar: str | None
    avatars: Sequence[ProfileAvatarReadModel]


@dataclass(frozen=True)
class ProfilePhotoResult:
    id: int | None
    album_id: int | None
    photo_url: str
    caption: str | None
    created_at: datetime


@dataclass(frozen=True)
class ProfilePhotoAlbumResult:
    id: int | None
    title: str
    kind: str
    photos: Sequence[ProfilePhotoResult]
    created_at: datetime | None = None


@dataclass(frozen=True)
class ProfilePhotosResult:
    user: User
    is_own_profile: bool
    albums: Sequence[ProfilePhotoAlbumResult]


@dataclass(frozen=True)
class PhotoAlbumResult:
    album: ProfilePhotoAlbumResult


@dataclass(frozen=True)
class FriendsResult:
    current_user: User
    friends: Sequence[UserReadModel]
    subscribers: Sequence[UserReadModel]
    subscriptions: Sequence[UserReadModel]


@dataclass(frozen=True)
class FriendActionResult:
    success: bool
    is_friend: bool
    is_subscribed: bool
    is_subscribed_to_current: bool
    message: str


@dataclass(frozen=True)
class AnalyticsSummaryResult:
    events_count: int
    unique_users: int
    users_registered: int
    posts_created: int
    comments_created: int
    likes_added: int
    likes_removed: int
    photos_deleted: int
