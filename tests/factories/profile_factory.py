from datetime import date, timedelta
from typing import Any, Callable, Optional

from faker import Faker

from backend.presentation.profiles.http.schemas import ProfileCreateRequest

fake = Faker("ru_RU")


def create_profile_create_data(
    first_name: Optional[str] = None,
    last_name: Optional[str] = None,
    middle_name: Optional[str] = None,
    birth_date: Optional[date] = None,
    gender: Optional[str] = None,
    phone_number: Optional[str] = None,
    country: Optional[str] = None,
    city: Optional[str] = None,
    street: Optional[str] = None,
    bio: Optional[str] = None,
) -> ProfileCreateRequest:
    """Генерирует случайные данные для ProfileCreate.

    Можно переопределять любые поля.
    """

    return ProfileCreateRequest(
        first_name=first_name or fake.first_name(),
        last_name=last_name or fake.last_name(),
        middle_name=middle_name or fake.first_name(),
        birth_date=birth_date or date.today() - timedelta(days=365 * 18),
        gender=gender or fake.random_element(elements=("Мужчина", "Женщина")),
        phone_number=phone_number or fake.phone_number(),
        country=country or fake.country(),
        city=city or fake.city(),
        street=street or fake.street_address(),
        bio=bio or fake.text(max_nb_chars=200),
    )


def _get_field_value(
    value: Optional[Any],
    default_generator: Callable,
    none_allowed: bool = True,
) -> Optional[Any]:
    """Вспомогательная функция для генерации значений полей."""
    if value is None and none_allowed:
        return None
    return value if value is not None else default_generator()


def create_profile_create_none_fields(
    first_name: Optional[str] = None,
    last_name: Optional[str] = None,
    middle_name: Optional[str] = None,
    birth_date: Optional[date] = None,
    gender: Optional[str] = None,
    phone_number: Optional[str] = None,
    country: Optional[str] = None,
    city: Optional[str] = None,
    street: Optional[str] = None,
    bio: Optional[str] = None,
) -> ProfileCreateRequest:
    """Генерирует случайные данные для ProfileCreate.

    Можно переопределять любые поля, включая явное указание None.
    """
    field_generators = {
        "first_name": fake.first_name,
        "last_name": fake.last_name,
        "middle_name": fake.first_name,
        "birth_date": lambda: date.today() - timedelta(days=365 * 18),
        "gender": lambda: fake.random_element(elements=("Мужчина", "Женщина")),
        "phone_number": fake.phone_number,
        "country": fake.country,
        "city": fake.city,
        "street": fake.street_address,
        "bio": lambda: fake.text(max_nb_chars=200),
    }

    return ProfileCreateRequest(
        first_name=_get_field_value(first_name, field_generators["first_name"]),
        last_name=_get_field_value(last_name, field_generators["last_name"]),
        middle_name=_get_field_value(middle_name, field_generators["middle_name"]),
        birth_date=_get_field_value(birth_date, field_generators["birth_date"]),
        gender=_get_field_value(gender, field_generators["gender"]),
        phone_number=_get_field_value(phone_number, field_generators["phone_number"]),
        country=_get_field_value(country, field_generators["country"]),
        city=_get_field_value(city, field_generators["city"]),
        street=_get_field_value(street, field_generators["street"]),
        bio=_get_field_value(bio, field_generators["bio"]),
    )
