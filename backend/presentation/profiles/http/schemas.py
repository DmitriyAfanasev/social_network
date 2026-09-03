import re
from datetime import date, datetime

from pydantic import BaseModel, Field, field_validator, model_validator

from backend.domain.user.entity.profile import PROFILE_VISIBILITY_POLICIES


class ProfileBaseRequest(BaseModel):
    first_name: str | None = None
    last_name: str | None = None
    middle_name: str | None = None
    birth_date: date | str | None = None
    gender: str | None = None
    phone_number: str | None = None
    country: str | None = None
    city: str | None = None
    street: str | None = None
    bio: str | None = None
    status: str | None = Field(default=None, max_length=140)
    profile_visibility: str | None = None
    friend_request_policy: str | None = None
    message_policy: str | None = None
    show_email: bool | None = None
    show_phone: bool | None = None
    show_birth_date: bool | None = None
    show_friends: bool | None = None
    show_posts: bool | None = None

    @field_validator("profile_visibility", "friend_request_policy", "message_policy")
    @classmethod
    def validate_privacy_policy(cls, value: str | None) -> str | None:
        if value is not None and value not in PROFILE_VISIBILITY_POLICIES:
            raise ValueError(f"Политика должна быть одной из: {', '.join(PROFILE_VISIBILITY_POLICIES)}")
        return value

    @field_validator("birth_date")
    @classmethod
    def validate_birth_date(cls, birth_date: date | str | None) -> date | None:
        if not birth_date:
            return None

        birth_date_str = birth_date.strftime("%Y-%m-%d") if isinstance(birth_date, date) else birth_date

        try:
            value = datetime.strptime(birth_date_str, "%Y-%m-%d").date()
        except ValueError as e:
            raise ValueError(f"Неверный формат даты: {birth_date}. Используйте YYYY-MM-DD.") from e

        today = date.today()
        age = today.year - value.year

        if value > today:
            raise ValueError("Дата рождения не может быть в будущем.")
        if value.year < 1900:
            raise ValueError("Дата рождения не может быть раньше 1900 года.")
        if age < 14:
            raise ValueError("Минимальный возраст регистрации — 14 лет.")

        return value

    @field_validator("status")
    @classmethod
    def normalize_status(cls, value: str | None) -> str | None:
        if value is None:
            return None
        normalized = value.strip()
        return normalized

    @model_validator(mode="after")
    def validate_phone_number(self) -> "ProfileBaseRequest":
        if not self.phone_number:
            return self

        cleaned_value = re.sub(r"[^\d]", "", self.phone_number)

        if len(cleaned_value) != 11:
            raise ValueError("Номер телефона должен содержать 11 цифр.")
        if cleaned_value.startswith("8"):
            formatted_number = f"+7{cleaned_value[1:]}"
        elif cleaned_value.startswith("7"):
            formatted_number = f"+{cleaned_value}"
        else:
            raise ValueError("Номер телефона должен начинаться с '8' или '7'.")

        self.phone_number = formatted_number
        return self


class ProfileCreateRequest(ProfileBaseRequest):
    pass


class ProfileUpdateRequest(ProfileBaseRequest):
    pass


class AvatarSelectRequest(BaseModel):
    avatar_url: str


class PhotoAlbumCreateRequest(BaseModel):
    title: str
