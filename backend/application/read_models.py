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
    avatar: str | None


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


class CommentReadModel(Protocol):
    id: int
    post_id: int
    user_id: int
    parent_id: int | None
    text: str
    user: UserReadModel | None
    created_at: datetime
    updated_at: datetime


class OutboxEventReadModel(Protocol):
    id: int
    event_type: str
    payload: JsonPayload
    attempts: int
    created_at: datetime
