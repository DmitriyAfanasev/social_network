import asyncio
from pathlib import Path
from urllib.parse import unquote

import aiofiles  # type: ignore[import-untyped]

from backend.application.ports.file_upload_service import FileUploadService, UploadFileSource
from backend.infra.storage.file_upload import (
    CHUNK_SIZE,
    FileKeyBuilder,
    FileUploadSettings,
    FileUploadValidator,
    create_parent_directory,
)


BASE_STATIC_DIR = "client_files"


class LocalFileUploadService(FileUploadService):
    def __init__(
        self,
        *,
        base_dir: str = BASE_STATIC_DIR,
        url_prefix: str = "/client_files",
        max_size_bytes: int,
    ) -> None:
        self.base_dir = Path(base_dir)
        self.url_prefix = url_prefix.rstrip("/")
        self.validator = FileUploadValidator(FileUploadSettings(max_size_bytes=max_size_bytes))
        self.key_builder = FileKeyBuilder()

    async def upload(
        self,
        *,
        file: UploadFileSource,
        directory: str,
    ) -> str:
        filename = self.validator.get_safe_filename(file.filename)
        self.validator.validate_declared_size(file.size)

        storage_key = self.key_builder.build(
            filename=filename,
            directory=directory,
        )
        file_path = self.base_dir / Path(storage_key)
        create_parent_directory(file_path)

        total_size = 0
        async with aiofiles.open(file_path, "wb") as buffer:
            while chunk := await file.read(CHUNK_SIZE):
                total_size += len(chunk)
                self.validator.validate_size(total_size)
                await buffer.write(chunk)

        self.validator.validate_size(total_size)
        return f"{self.url_prefix}/{storage_key}"

    async def delete(self, *, file_url: str) -> bool:
        storage_key = self._storage_key_from_url(file_url)
        if storage_key is None:
            return False

        base_dir = self.base_dir.resolve()
        file_path = (base_dir / storage_key).resolve()
        if base_dir != file_path and base_dir not in file_path.parents:
            raise ValueError("Некорректный путь к файлу")

        if not await asyncio.to_thread(file_path.exists):
            return True

        try:
            await asyncio.to_thread(file_path.unlink)
        except FileNotFoundError:
            return True
        return True

    def _storage_key_from_url(self, file_url: str) -> Path | None:
        prefix = f"{self.url_prefix}/"
        if not file_url.startswith(prefix):
            return None

        storage_key = unquote(file_url.removeprefix(prefix)).strip("/")
        if not storage_key:
            return None

        return Path(storage_key)
