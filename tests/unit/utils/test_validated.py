import pytest
from backend.shared.utils.validated import validate_username


@pytest.mark.parametrize(
    "username",
    [
        "valid_username",  # стандартный случай
        "Valid123",  # с цифрами
        "user_name",  # с подчеркиванием
        "UPPER_CASE",  # заглавные буквы
        "_underscore_",  # подчеркивания по краям
        "123456",  # из цифр
    ],
)
def test_valid_usernames(username: str) -> None:
    """Тестирует допустимые имена пользователей"""
    assert validate_username(username) == username


@pytest.mark.parametrize(
    "username, error_msg",
    [
        (
            "invalid username",
            "Username can only contain letters, numbers and underscores",
        ),  # пробел
        (
            "  invalid_username  ",
            "Username can only contain letters, numbers and underscores",
        ),  # пробел
        (
            "user@name",
            "Username can only contain letters, numbers and underscores",
        ),  # спецсимвол
        (
            "user-name",
            "Username can only contain letters, numbers and underscores",
        ),  # дефис
        (
            "",
            "Username can only contain letters, numbers and underscores",
        ),  # пустая строка
        (
            "user.name",
            "Username can only contain letters, numbers and underscores",
        ),  # точка
        (
            "user/name",
            "Username can only contain letters, numbers and underscores",
        ),  # слэш
    ],
)
def test_invalid_usernames(username: str, error_msg: str) -> None:
    """Тестирует недопустимые имена пользователей"""
    with pytest.raises(ValueError) as exc_info:
        validate_username(username)
    assert error_msg in str(exc_info.value)
