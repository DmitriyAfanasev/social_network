from dataclasses import dataclass
from typing import Protocol

from backend.application.ports.file_upload_service import UploadFileSource


@dataclass(frozen=True, slots=True)
class StoredMedia:
    """Метаданные файла, сохранённого в объектном хранилище."""

    id: int
    filename: str
    content_type: str | None
    size: int


class MediaStorage(Protocol):
    """Порт для загрузки, чтения и удаления медиафайлов."""

    async def upload(
        self, *, file: UploadFileSource, directory: str, uploaded_by: int | None = None
    ) -> StoredMedia:
        """Сохраняет файл и возвращает его идентификатор и метаданные."""

    async def get(self, media_id: int) -> tuple[StoredMedia, bytes] | None:
        """Возвращает метаданные и содержимое файла по идентификатору."""

    async def get_default_avatar(self) -> tuple[StoredMedia, bytes] | None:
        """Возвращает системный дефолтный аватар."""

    async def delete(self, media_id: int) -> bool:
        """Удаляет запись медиа и объект из хранилища."""
