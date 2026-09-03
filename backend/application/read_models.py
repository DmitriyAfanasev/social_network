from collections.abc import Sequence
from datetime import date, datetime
from typing import Protocol

from backend.application.events import JsonPayload


class ProfileReadModel(Protocol):
    first_name: str | None
    last_name: str | None
    middle_name: str | None
    full_name: str | None
    birth_date: date | None
    gender: str | None
    phone_number: str | None
    country: str | None
    city: str | None
    street: str | None
    bio: str | None
    status: str | None
    avatar: str | None
    profile_visibility: str
    friend_request_policy: str
    message_policy: str
    show_email: bool
    show_phone: bool
    show_birth_date: bool
    show_friends: bool
    show_posts: bool


class ProfileAvatarReadModel(Protocol):
    id: int
    avatar_url: str
    created_at: datetime


class ProfilePhotoReadModel(Protocol):
    id: int
    album_id: int
    user_id: int
    photo_url: str
    caption: str | None
    created_at: datetime


class ProfilePhotoAlbumReadModel(Protocol):
    id: int
    user_id: int
    title: str
    created_at: datetime
    photos: Sequence[ProfilePhotoReadModel]


class UserReadModel(Protocol):
    id: int
    username: str
    email: str
    is_active: bool
    is_superuser: bool
    created_at: datetime
    profile: ProfileReadModel | None


class PostReadModel(Protocol):
    id: int
    content: str | None
    image: str | None
    author_id: int
    author: UserReadModel | None
    created_at: datetime
    updated_at: datetime
    preview_comment: "CommentReadModel | None"
    image_content_type: str | None


class CommentReadModel(Protocol):
    id: int
    post_id: int
    user_id: int
    parent_id: int | None
    text: str
    user: UserReadModel | None
    created_at: datetime
    updated_at: datetime
    likes_count: int


class MediaReadModel(Protocol):
    content_type: str | None


class MessageReadModel(Protocol):
    id: int
    conversation_id: int
    sender_id: int
    text: str
    created_at: datetime
    media_id: int | None
    media: MediaReadModel | None
    edited_at: datetime | None
    deleted_at: datetime | None


class ConversationParticipantReadModel(Protocol):
    user_id: int
    archived_at: datetime | None
    pinned_at: datetime | None
    muted_at: datetime | None


class ConversationReadModel(Protocol):
    id: int
    created_at: datetime
    participants: Sequence[ConversationParticipantReadModel]
    last_message: MessageReadModel | None


class OutboxEventReadModel(Protocol):
    id: int
    event_type: str
    payload: JsonPayload
    attempts: int
    created_at: datetime
