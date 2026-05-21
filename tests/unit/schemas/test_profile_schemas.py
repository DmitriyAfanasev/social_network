from datetime import date, timedelta

import pytest

from backend.presentation.profiles.http.schemas import (
    ProfileCreateRequest,
    ProfileUpdateRequest,
)
from tests.factories.profile_factory import (
    create_profile_create_data,
)


class TestProfileCreate:
    """Группа тестов для схемы ProfileCreate"""

    def test_profile_create_valid(self) -> None:
        profile = create_profile_create_data()
        assert profile is not None
        assert profile.first_name is not None
        assert profile.last_name is not None
        assert profile.middle_name is not None
        assert profile.gender in ("Мужчина", "Женщина")
        assert isinstance(profile, ProfileCreateRequest)
        assert len(profile.phone_number) == 12
        assert profile.birth_date.year >= 1900
        assert profile.birth_date <= date.today()
        assert len(profile.bio) <= 200

    def test_profile_create_with_custom_values(self) -> None:
        custom_data = create_profile_create_data(
            first_name="Александр",
            birth_date=date(2000, 1, 1),
            phone_number="89991234567",
        )
        assert custom_data.first_name == "Александр"
        assert custom_data.birth_date == date(2000, 1, 1)
        assert custom_data.phone_number == "+79991234567"

    def test_profile_create_invalid_birth_date_future(self) -> None:
        with pytest.raises(ValueError) as exc_info:
            _ = create_profile_create_data(birth_date=date(date.today().year + 1, 1, 1))

        assert "не может быть в будущем" in str(exc_info.value)

    def test_profile_create_invalid_birth_date_too_young(self) -> None:
        too_young = date.today().replace(year=date.today().year - 13)
        with pytest.raises(ValueError) as exc_info:
            _ = create_profile_create_data(birth_date=too_young)

        assert "Минимальный возраст регистрации — 14 лет." in str(exc_info.value)

    def test_profile_create_invalid_birth_date_before_1900(self) -> None:
        with pytest.raises(ValueError) as exc_info:
            _ = create_profile_create_data(birth_date=date(1899, 1, 1))

        assert "раньше 1900 года" in str(exc_info.value)

    def test_profile_create_invalid_phone_number_length(self) -> None:
        with pytest.raises(ValueError) as exc_info:
            _ = create_profile_create_data(phone_number="123456789")

        assert "должен содержать 11 цифр" in str(exc_info.value)

    def test_profile_create_invalid_phone_number_prefix(self) -> None:
        with pytest.raises(ValueError) as exc_info:
            _ = create_profile_create_data(phone_number="69991234567")

        assert "должен начинаться с '8' или '7'" in str(exc_info.value)


class TestProfileUpdate:
    """Тесты для схемы обновления профиля ProfileUpdate."""

    @pytest.mark.parametrize(
        "field, value, expected",
        [
            ("first_name", "Иван", "Иван"),
            ("last_name", "Иванов", "Иванов"),
            ("middle_name", "Иванович", "Иванович"),
            ("gender", "Мужчина", "Мужчина"),
            ("birth_date", date(2000, 1, 1), date(2000, 1, 1)),
            ("phone_number", "89991234567", "+79991234567"),
            ("phone_number", "79123456789", "+79123456789"),
            ("country", "Россия", "Россия"),
            ("city", "Москва", "Москва"),
            ("street", "Ленина 1", "Ленина 1"),
            ("bio", "Тестовая биография", "Тестовая биография"),
        ],
    )
    def test_update_individual_fields(
        self, field: str, value: str | date, expected: str | date
    ) -> None:
        """Тест валидации отдельных полей при обновлении."""
        update_data = {field: value}
        profile = ProfileUpdateRequest(**update_data)
        assert getattr(profile, field) == expected

    def test_partial_update(self) -> None:
        """Тест частичного обновления профиля."""
        update_data = {"first_name": "Петр", "phone_number": "79123456789"}
        profile = ProfileUpdateRequest(**update_data)
        assert profile.first_name == "Петр"
        assert profile.phone_number == "+79123456789"
        assert profile.last_name is None

    @pytest.mark.parametrize(
        "birth_date",
        [
            "2000-01-01",
            date(2000, 1, 1),
        ],
    )
    def test_valid_birth_date_formats(self, birth_date: date) -> None:
        """Тест допустимых форматов даты рождения."""
        profile = ProfileUpdateRequest(birth_date=birth_date)
        assert profile.birth_date == date(2000, 1, 1)

    @pytest.mark.parametrize(
        "birth_date, error_msg",
        [
            (date.today() + timedelta(days=1), "не может быть в будущем"),
            ("1899-12-31", "раньше 1900 года"),
            (
                date.today().replace(year=date.today().year - 13),
                "Минимальный возраст регистрации — 14 лет",
            ),
            ("invalid-date", "Неверный формат даты"),
        ],
    )
    def test_invalid_birth_dates(self, birth_date: date, error_msg: str) -> None:
        """Тест недопустимых дат рождения."""
        with pytest.raises(ValueError) as exc_info:
            ProfileUpdateRequest(birth_date=birth_date)
        assert error_msg in str(exc_info.value)

    @pytest.mark.parametrize(
        "phone_number, expected",
        [
            ("89991234567", "+79991234567"),
            ("79123456789", "+79123456789"),
            ("+79123456789", "+79123456789"),
            ("8 (999) 123-45-67", "+79991234567"),
        ],
    )
    def test_valid_phone_numbers(self, phone_number: str, expected: str) -> None:
        """Тест валидных номеров телефона."""
        profile = ProfileUpdateRequest(phone_number=phone_number)
        assert profile.phone_number == expected

    @pytest.mark.parametrize(
        "phone_number,error_msg",
        [
            ("123456789", "должен содержать 11 цифр"),
            ("59991234567", "должен начинаться с '8' или '7'"),
            ("8abc1234567", "должен содержать 11 цифр"),
        ],
    )
    def test_invalid_phone_numbers(self, phone_number: str, error_msg: str) -> None:
        """Тест недопустимых номеров телефона."""
        with pytest.raises(ValueError) as exc_info:
            ProfileUpdateRequest(phone_number=phone_number)
        assert error_msg in str(exc_info.value)

    def test_clear_fields_with_none(self) -> None:
        """Тест очистки полей путем установки None."""
        profile = ProfileUpdateRequest(first_name=None, phone_number=None, bio=None)
        assert profile.first_name is None
        assert profile.phone_number is None
        assert profile.bio is None

    def test_multiple_fields_update(self) -> None:
        """Тест одновременного обновления нескольких полей."""
        update_data = {
            "first_name": "Сергей",
            "last_name": "Петров",
            "birth_date": "1995-05-15",
            "phone_number": "89123456789",
            "bio": "Новая биография",
        }
        profile = ProfileUpdateRequest(**update_data)
        assert profile.first_name == "Сергей"
        assert profile.last_name == "Петров"
        assert profile.birth_date == date(1995, 5, 15)
        assert profile.phone_number == "+79123456789"
        assert profile.bio == "Новая биография"

    def test_empty_update(self) -> None:
        """Тест пустого обновления (все поля None)."""
        profile = ProfileUpdateRequest()
        assert profile.first_name is None
        assert profile.last_name is None
        assert profile.birth_date is None
