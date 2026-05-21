import os
import posixpath
import uuid
from dataclasses import dataclass
from pathlib import Path

from backend.application.ports.file_upload_service import UploadFileSource


CHUNK_SIZE = 1024 * 1024


@dataclass(frozen=True, slots=True)
class FileUploadSettings:
    max_size_bytes: int


class FileUploadValidator:
    def __init__(self, settings: FileUploadSettings) -> None:
        self.settings = settings

    def validate_declared_size(self, file_size: int | None) -> None:
        if file_size is None:
            return
        self.validate_size(file_size)

    def validate_size(self, file_size: int) -> None:
        if file_size <= 0:
            raise ValueError("Файл должен быть больше 0 байт.")

        if file_size > self.settings.max_size_bytes:
            max_mb = self.settings.max_size_bytes // (1024 * 1024)
            raise ValueError(f"Размер файла превышает максимально допустимый ({max_mb} МБ).")

    def get_safe_filename(self, filename: str | None) -> str:
        safe_filename = Path(filename or "").name
        if not safe_filename:
            raise ValueError("У файла отсутствует имя.")
        return safe_filename


class FileKeyBuilder:
    def build(self, *, filename: str, directory: str) -> str:
        normalized_path = directory.strip("/").strip()
        unique_filename = f"{uuid.uuid4()}_{filename}"
        return posixpath.join(normalized_path, unique_filename)


async def read_upload_file(file: UploadFileSource, validator: FileUploadValidator) -> bytes:
    validator.validate_declared_size(file.size)

    total_size = 0
    content = bytearray()
    while chunk := await file.read(CHUNK_SIZE):
        total_size += len(chunk)
        validator.validate_size(total_size)
        content.extend(chunk)

    validator.validate_size(total_size)
    return bytes(content)


def create_parent_directory(file_path: Path) -> None:
    os.makedirs(file_path.parent, exist_ok=True)
