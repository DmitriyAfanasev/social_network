from typing import TYPE_CHECKING
import pytest

from tests.factories.profile_factory import (
    create_profile_create_data,
    create_profile_create_none_fields,
)
from tests.factories.content_factory import ContentCreateFactory

if TYPE_CHECKING:
    from backend.presentation.shared.http.schemas.content import ContentCreate
    from backend.presentation.profiles.http.schemas import ProfileCreateRequest


@pytest.fixture
def profile_create_data() -> "ProfileCreateRequest":
    return create_profile_create_data()


@pytest.fixture
def profile_create_none_fields() -> "ProfileCreateRequest":
    return create_profile_create_none_fields()


@pytest.fixture
def valid_content_data() -> "ContentCreate":
    """Возвращает валидный объект ContentCreate."""
    return ContentCreateFactory.build()


@pytest.fixture
def valid_content_with_only_text() -> "ContentCreate":
    return ContentCreateFactory.build_only_text()


@pytest.fixture
def valid_content_with_only_image() -> "ContentCreate":
    return ContentCreateFactory.build_only_image()
