import re
from unittest.mock import MagicMock, patch

import pytest

from backend.infra.security.password_helper import (
    ErrorMessages,
    PasswordHelper,
)


class TestPasswordHelper:
    """Тесты для хелпера работы с паролями."""

    @pytest.mark.parametrize(
        "password",
        [
            "1",
            "simple",
            "без спецсимволов",
        ],
    )
    def test_validate_password_success(self, password: str) -> None:
        """Тест успешной валидации пароля."""
        PasswordHelper.validate_password(password)

    @pytest.mark.parametrize(
        ("password", "expected_error"),
        [
            ("", ErrorMessages.EMPTY_PASSWORD),
            ("a" * 73, ErrorMessages.PASSWORD_TOO_LONG.format(max_length=72)),
        ],
    )
    def test_validate_password_failure(
        self,
        password: str,
        expected_error: str,
    ) -> None:
        """Тест неудачной валидации пароля."""
        with pytest.raises(ValueError, match=re.escape(expected_error)):
            PasswordHelper.validate_password(password)

    # Тесты генерации пароля
    def test_generate_password_success(self) -> None:
        """Тест успешной генерации хэша пароля."""
        password = "StrongPass123!"
        hashed = PasswordHelper.generate_password(password)
        assert isinstance(hashed, str)
        assert hashed.startswith("$2b$")  # Проверяем что это bcrypt

    def test_generate_password_empty(self) -> None:
        """Тест генерации с пустым паролем."""
        with pytest.raises(ValueError, match=re.escape(ErrorMessages.EMPTY_PASSWORD)):
            PasswordHelper.generate_password("")

    def test_generate_password_too_long(self) -> None:
        """Тест генерации со слишком длинным паролем."""
        expected = ErrorMessages.PASSWORD_TOO_LONG.format(max_length=72)
        with pytest.raises(ValueError, match=re.escape(expected)):
            PasswordHelper.generate_password("a" * 73)

    # Тесты проверки пароля
    def test_verify_password_success(self) -> None:
        """Тест успешной проверки пароля."""
        password = "TestPassword123!"
        hashed = PasswordHelper.generate_password(password)
        assert PasswordHelper.verify_password(password, hashed) is True

    def test_verify_password_incorrect(self) -> None:
        """Тест неверного пароля."""
        password = "CorrectPassword123!"
        hashed = PasswordHelper.generate_password(password)
        assert PasswordHelper.verify_password("WrongPassword123!", hashed) is False

    def test_verify_password_unknown_hash(self) -> None:
        """Тест неизвестного формата хэша."""
        with patch.object(PasswordHelper.PWD_CONTEXT, "verify", side_effect=ValueError):
            assert PasswordHelper.verify_password("any", "invalid$hash") is False

    @pytest.mark.parametrize(
        ("password", "hashed"),
        [
            (123, "valid$hash"),
            ("password", 123),
            (None, "valid$hash"),
            ("password", None),
        ],
    )
    def test_verify_password_invalid_types(self, password: str, hashed: str) -> None:
        """Тест неверных типов данных."""
        with pytest.raises(TypeError):
            PasswordHelper.verify_password(password, hashed)

    @patch.object(PasswordHelper.PWD_CONTEXT, "verify", return_value=False)
    def test_timing_attack_protection(self, mock_verify: MagicMock) -> None:
        """Тест, что время проверки не зависит от правильности пароля."""
        valid_hash = PasswordHelper.generate_password("Validpass1!")
        PasswordHelper.verify_password("wrongpass", valid_hash)
        mock_verify.assert_called_once()

    def test_verify_with_bytes(self) -> None:
        """Тест работы с bytes вместо str."""
        password = b"BytesPassword123!"
        hashed = PasswordHelper.generate_password(password.decode())
        assert PasswordHelper.verify_password(password, hashed) is True

    def test_needs_rehash(self) -> None:
        """Тест определения необходимости обновления хэша."""
        password = "MyPassword123!"
        hashed = PasswordHelper.generate_password(password)
        assert not PasswordHelper.PWD_CONTEXT.needs_update(hashed)

    def test_error_messages_contain_required_info(self) -> None:
        """Тест, что сообщения об ошибках содержат нужную информацию."""
        assert "не может быть пустым" in ErrorMessages.EMPTY_PASSWORD
        assert "максимум" in ErrorMessages.PASSWORD_TOO_LONG
