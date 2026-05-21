import factory
from faker import Faker
from faker.providers import BaseProvider

from backend.presentation.shared.http.schemas.content import ContentCreate


class InvalidImageExtensionProvider(BaseProvider):
    """
    Кастомный провайдер, что создания не валидных расширений фотографий.
    """

    def invalid_image_extension(self) -> str:
        """Возвращает случайное или фиксированное невалидное расширение."""
        return self.random_element(["sh", "exe", "txt", "pdf", "mp3", "mp4", "avi"])

    def invalid_image_filename(self) -> str:
        """Возвращает невалидное имя файла."""
        extension = self.invalid_image_extension()
        return f"invalid_image.{extension}"


fake = Faker()
fake.add_provider(InvalidImageExtensionProvider)


class ContentCreateFactory(factory.Factory):
    """Фабрика для создания тестовых объектов ContentCreate."""

    class Meta:
        model = ContentCreate

    content = factory.Faker("text", max_nb_chars=200)
    image = factory.Faker(
        "file_name", extension=fake.random_element(elements=("jpg", "png"))
    )
    author_id = factory.Faker("random_int", min=1, max=9999)

    @classmethod
    def build_only_text(cls, **kwargs) -> ContentCreate:
        """Создает контент только с текстом (без изображения)."""
        return cls.build(image=None, **kwargs)

    @classmethod
    def build_only_image(cls, **kwargs) -> ContentCreate:
        """Создает контент только с изображением (без текста)."""
        return cls.build(content=None, **kwargs)

    @classmethod
    def build_only_image_high(cls, **kwargs) -> ContentCreate:
        """Создает контент только с изображением (без текста)."""
        return cls.build(content=None, image=fake.image(size=5001), **kwargs)

    @classmethod
    def build_invalid_empty(cls, **kwargs) -> ContentCreate:
        """
        Возвращает невалидные данные (без контента и изображения).
        Не создает объект, так как это вызовет ValidationError.
        """
        return cls.build(content=None, image=None, **kwargs)

    @classmethod
    def build_with_invalid_image_extension(cls, **kwargs) -> ContentCreate:
        """Возвращает с невалидным форматом фотографии."""
        return cls.build(image=fake.invalid_image_filename(), **kwargs)
