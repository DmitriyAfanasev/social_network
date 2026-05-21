import pytest
from pydantic import ValidationError

from backend.presentation.shared.http.schemas.content import ContentCreate
from tests.factories.content_factory import ContentCreateFactory


class TestContentCreateSchema:
    """
    Группа тестов для ContentCreate — бизнес-валидации и корректное создание модели.
    Проверяет:
        - обязательность author_id
        - валидацию content и image
        - возможность создания поста только с текстом или только с изображением
        - обработку крайних случаев (пустой контент, невалидный image, отсутствие author_id)
    """

    def test_valid_create(
        self,
        valid_content_data: ContentCreate,
    ) -> None:
        """
        Проверяет успешное создание поста со всеми валидными полями.
        """
        assert isinstance(valid_content_data, ContentCreate)
        assert valid_content_data.content is not None
        assert isinstance(valid_content_data.content, str)
        assert valid_content_data.image is not None
        assert isinstance(valid_content_data.content, str)
        assert isinstance(valid_content_data.author_id, int)
        assert valid_content_data.author_id > 0

    def test_create_with_only_text(
        self,
        valid_content_with_only_text: ContentCreate,
    ) -> None:
        """
        Проверяет, что можно создать пост только с текстом.
        """
        assert valid_content_with_only_text.content is not None
        assert isinstance(valid_content_with_only_text.content, str)
        assert valid_content_with_only_text.image is None
        assert valid_content_with_only_text.author_id > 0

    def test_create_with_only_image(
        self,
        valid_content_with_only_image: ContentCreate,
    ) -> None:
        """
        Проверяет, что можно создать пост только с изображением.
        """
        assert valid_content_with_only_image.image is not None
        assert isinstance(valid_content_with_only_image.image, str)
        assert valid_content_with_only_image.content is None
        assert valid_content_with_only_image.author_id > 0

    def test_create_with_empty_content_and_image_raises_error(self) -> None:
        """
        Проверяет, что нельзя создать пост без контента и изображения.
        """
        with pytest.raises(ValidationError) as exc_info:
            ContentCreate(content=None, image=None, author_id=1)

        assert "Пост должен содержать текст или изображение" in str(exc_info.value)

    @pytest.mark.parametrize(
        "text_for_content, path_to_file_image, expected_error_message",
        [
            ("   ", None, "Пост должен содержать текст или изображение"),
            (None, "   ", "Некорректное расширение файла изображения"),
            (None, "some_path/image.mp4", "Некорректное расширение файла изображения"),
            (None, "some_path/image.mp3", "Некорректное расширение файла изображения"),
        ],
    )
    def test_create_with_blank_content_raises_error(
        self,
        text_for_content: str,
        path_to_file_image: str,
        expected_error_message: str,
    ) -> None:
        """
        Проверяет, что пустой/пробельный контент вызывает ошибку.
        """
        with pytest.raises(ValidationError) as exc_info:
            ContentCreate(
                content=text_for_content, image=path_to_file_image, author_id=1
            )

        assert expected_error_message in str(exc_info.value)

    @pytest.mark.parametrize(
        "author_id, message_error",
        [
            (-1, "Input should be greater than 0"),
            (0, "Input should be greater than 0"),
            (-1000, "Input should be greater than 0"),
            (2.4, "Input should be a valid integer"),
            (-2.4, "Input should be a valid integer"),
        ],
    )
    def test_create_with_missing_author_id_raises_error(
        self,
        valid_content_data: ContentCreate,
        author_id: int,
        message_error: str,
    ) -> None:
        """
        Проверяет, что author_id обязателен при создании поста.
        """
        with pytest.raises(ValidationError) as exc_info:
            ContentCreate(
                content=valid_content_data.content,
                image=valid_content_data.image,
                author_id=author_id,
            )

        assert message_error in str(exc_info.value)

    def test_create_with_invalid_image_extension_raises_error(self) -> None:
        """
        Проверяет, что неподдерживаемое расширение файла вызывает ошибку.
        """
        with pytest.raises(ValidationError) as exc_info:
            [
                ContentCreateFactory.build_with_invalid_image_extension()
                for _ in range(10)
            ]

        assert "Некорректное расширение файла изображения" in str(exc_info.value)
