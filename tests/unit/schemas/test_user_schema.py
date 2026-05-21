import pytest
from datetime import datetime
from pydantic import ValidationError

from backend.presentation.users.http.schemas import (
    UserBaseRequest,
    UserCreateRequest,
    UserUpdateRequest,
    UserResponse,
)


class TestUserBase:
    """Тесты для базовой схемы пользователя."""

    def test_valid_user_base(self) -> None:
        data = {"username": "valid_user", "email": "valid@example.com"}
        user = UserBaseRequest(**data)
        assert user.username == "valid_user"
        assert user.email == "valid@example.com"

    def test_invalid_email(self) -> None:
        with pytest.raises(ValidationError):
            UserBaseRequest(username="valid", email="invalid-email")
            UserBaseRequest(username="aa", email="valid@example.com")
            UserBaseRequest(username="AA" * 10, email="valid@example.com")


class TestUserCreate:
    """Тесты для схемы создания пользователя."""

    @pytest.mark.parametrize(
        "username",
        ["valid", "valid_username", "valid123", "a" * 20],  # Максимальная длина
    )
    def test_valid_usernames(self, username) -> None:
        data = {
            "username": username,
            "password": "12345678",
            "email": "test@example.com",
        }
        assert UserCreateRequest(**data)

    @pytest.mark.parametrize(
        "username",
        ["sh", "a" * 21, "invalid!", "with space"],
    )
    def test_invalid_usernames(self, username: str) -> None:
        with pytest.raises(ValidationError):
            UserCreateRequest(
                username=username, password="secure123", email="test@example.com"
            )

    @pytest.mark.parametrize(
        "password",
        ["12345678", "long_password_secure", "with!@#$%^&*"],
    )
    def test_valid_passwords(self, password) -> None:
        data = {
            "username": "valid_user",
            "password": password,
            "email": "test@example.com",
        }
        assert UserCreateRequest(**data)

    def test_password_too_short(self) -> None:
        with pytest.raises(ValidationError) as exc_info:
            UserCreateRequest(
                username="valid_user", password="short", email="test@example.com"
            )
        assert "at least 8 characters" in str(exc_info.value)

    def test_with_profile(self) -> None:
        profile_data = {
            "first_name": "John",
            "last_name": "Doe",
            "birth_date": "2000-01-01",
        }
        data = {
            "username": "valid_user",
            "password": "secure123",
            "email": "test@example.com",
            "profile": profile_data,
        }
        user = UserCreateRequest(**data)
        assert user.profile is not None
        assert user.profile.first_name == "John"


class TestUserUpdate:
    """Тесты для схемы обновления пользователя."""

    def test_partial_update(self) -> None:
        data = {"username": "new_username"}
        user_update = UserUpdateRequest(**data)
        assert user_update.username == "new_username"
        assert user_update.password is None
        assert user_update.email is None

    def test_all_fields_update(self) -> None:
        data = {
            "username": "new_username",
            "password": "new_password123",
            "email": "new@example.com",
        }
        user_update = UserUpdateRequest(**data)
        assert user_update.username == "new_username"
        assert user_update.password == "new_password123"
        assert user_update.email == "new@example.com"

    @pytest.mark.parametrize("username", ["in", "a" * 21, "invalid!"])
    def test_invalid_usernames(self, username) -> None:
        with pytest.raises(ValidationError):
            UserUpdateRequest(username=username)

    def test_invalid_password_length(self) -> None:
        with pytest.raises(ValidationError):
            UserUpdateRequest(password="short")


class TestUserResponse:
    """Тесты для схемы ответа с пользователем."""

    def test_response_model(self) -> None:
        data = {
            "id": 1,
            "username": "test_user",
            "email": "test@example.com",
            "is_active": True,
            "created_at": datetime.now(),
        }
        user_response = UserResponse(**data)
        assert user_response.id == 1
        assert user_response.username == "test_user"
        assert user_response.email == "test@example.com"
        assert user_response.is_active is True
        assert isinstance(user_response.created_at, datetime)

    def test_default_is_active(self) -> None:
        data = {
            "id": 1,
            "username": "test_user",
            "email": "test@example.com",
            "created_at": datetime.now(),
        }
        user_response = UserResponse(**data)
        assert user_response.is_active is True
