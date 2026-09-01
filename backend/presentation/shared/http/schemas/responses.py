from datetime import date, datetime

from pydantic import BaseModel, EmailStr, Field


class ProfileResponse(BaseModel):
    first_name: str | None = None
    last_name: str | None = None
    middle_name: str | None = None
    full_name: str | None = None
    birth_date: date | None = None
    gender: str | None = None
    phone_number: str | None = None
    country: str | None = None
    city: str | None = None
    street: str | None = None
    bio: str | None = None
    avatar: str | None = None


class UserResponse(BaseModel):
    id: int
    username: str
    email: EmailStr | None = None
    is_active: bool
    is_superuser: bool
    created_at: datetime
    profile: ProfileResponse | None = None


class PostResponse(BaseModel):
    id: int
    content: str | None = None
    image: str | None = None
    author_id: int
    author: UserResponse | None = None
    likes_count: int = 0
    comments_count: int = 0
    is_liked_by_current: bool = False
    liked_user_ids: list[int] = Field(default_factory=list)
    liked_users: list[UserResponse] = Field(default_factory=list)
    created_at: datetime
    updated_at: datetime


class CommentResponse(BaseModel):
    id: int
    post_id: int
    user_id: int
    parent_id: int | None = None
    text: str
    author: UserResponse | None = None
    created_at: datetime
    updated_at: datetime


class MessageResponse(BaseModel):
    message: str


class AuthResponse(BaseModel):
    user: UserResponse | None


class FeedResponse(BaseModel):
    current_user: UserResponse | None
    posts: list[PostResponse]
    page: int
    total_pages: int


class PostEnvelopeResponse(BaseModel):
    post: PostResponse


class RemovePostImageResponse(BaseModel):
    success: bool
    message: str


class CommentsPageResponse(BaseModel):
    comments: list[CommentResponse]
    has_more: bool
    offset: int
    limit: int
    post_id: int


class ToggleLikeResponse(BaseModel):
    success: bool
    action: str
    likes_count: int
    liked: bool


class ProfilePageResponse(BaseModel):
    user: UserResponse | None
    is_own_profile: bool
    is_friend: bool = False
    is_subscribed: bool = False
    is_subscribed_to_current: bool = False
    current_user: UserResponse | None
    posts: list[PostResponse]


class UserEnvelopeResponse(BaseModel):
    user: UserResponse | None


class AvatarUploadResponse(BaseModel):
    message: str
    avatar_url: str


class AvatarRemoveResponse(BaseModel):
    new_avatar: str
    message: str


class AvatarHistoryItemResponse(BaseModel):
    id: int
    avatar_url: str
    created_at: datetime
    is_current: bool = False


class AvatarHistoryResponse(BaseModel):
    current_avatar: str | None = None
    avatars: list[AvatarHistoryItemResponse]


class ProfilePhotoResponse(BaseModel):
    id: int | None = None
    album_id: int | None = None
    photo_url: str
    caption: str | None = None
    created_at: datetime


class ProfilePhotoAlbumResponse(BaseModel):
    id: int | None = None
    title: str
    kind: str
    created_at: datetime | None = None
    photos: list[ProfilePhotoResponse]


class ProfilePhotosResponse(BaseModel):
    user: UserResponse | None
    is_own_profile: bool
    albums: list[ProfilePhotoAlbumResponse]


class PhotoAlbumEnvelopeResponse(BaseModel):
    album: ProfilePhotoAlbumResponse


class FriendsResponse(BaseModel):
    current_user: UserResponse | None
    friends: list[UserResponse]
    subscribers: list[UserResponse]
    subscriptions: list[UserResponse]


class FriendActionResponse(BaseModel):
    success: bool
    is_friend: bool
    is_subscribed: bool
    is_subscribed_to_current: bool
    message: str
