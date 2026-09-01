from dataclasses import dataclass
from datetime import date

from backend.application.ports.file_upload_service import UploadFileSource


@dataclass(frozen=True)
class LoginCommand:
    email: str
    password: str


@dataclass(frozen=True)
class RegistrationConfirmationCommand:
    email: str
    base_url: str


@dataclass(frozen=True)
class ConfirmRegistrationCommand:
    token: str


@dataclass(frozen=True)
class RegisterUserCommand:
    username: str
    email: str
    password: str
    password2: str
    base_url: str


@dataclass(frozen=True)
class PasswordResetRequestCommand:
    email: str
    base_url: str


@dataclass(frozen=True)
class PasswordResetCommand:
    token: str
    new_password: str
    confirm_password: str


@dataclass(frozen=True)
class ProfileUpdateCommand:
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
    profile_visibility: str | None = None
    friend_request_policy: str | None = None
    message_policy: str | None = None
    show_email: bool | None = None
    show_phone: bool | None = None
    show_birth_date: bool | None = None
    show_friends: bool | None = None
    show_posts: bool | None = None


@dataclass(frozen=True)
class CreatePostCommand:
    content: str | None
    image: UploadFileSource | None = None


@dataclass(frozen=True)
class UpdatePostCommand:
    content: str | None
    image: UploadFileSource | None = None


@dataclass(frozen=True)
class CreateCommentCommand:
    content: str
    parent_id: int | None = None


@dataclass(frozen=True)
class UpdateCommentCommand:
    content: str
