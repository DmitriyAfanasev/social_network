import asyncio
import hashlib
import importlib
import posixpath
from typing import Any

from pydantic import SecretStr
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from backend.application.ports.file_upload_service import UploadFileSource
from backend.application.ports.media_storage import MediaStorage, StoredMedia
from backend.infra.config import DEFAULT_AVATAR_OBJECT_KEY
from backend.infra.models.sqlalchemy import Media
from backend.infra.storage.file_upload import (
    FileKeyBuilder,
    FileUploadSettings,
    FileUploadValidator,
    read_upload_file,
)


class S3MediaStorage(MediaStorage):
    """Хранит бинарные данные в S3/MinIO, а метаданные — в PostgreSQL."""

    def __init__(
        self,
        session: AsyncSession,
        *,
        bucket_name: str,
        max_size_bytes: int,
        region_name: str | None = None,
        endpoint_url: str | None = None,
        access_key_id: SecretStr | None = None,
        secret_access_key: SecretStr | None = None,
        key_prefix: str = "",
    ) -> None:
        if not bucket_name:
            raise ValueError("S3 bucket_name is required")
        self.session = session
        self.bucket_name = bucket_name
        self.region_name = region_name
        self.endpoint_url = endpoint_url
        self.access_key_id = access_key_id
        self.secret_access_key = secret_access_key
        self.key_prefix = key_prefix.strip("/")
        self.validator = FileUploadValidator(FileUploadSettings(max_size_bytes=max_size_bytes))
        self.key_builder = FileKeyBuilder()

    async def upload(
        self, *, file: UploadFileSource, directory: str, uploaded_by: int | None = None
    ) -> StoredMedia:
        """Загружает файл в S3 и создаёт запись метаданных."""
        filename = self.validator.get_safe_filename(file.filename)
        content = await read_upload_file(file, self.validator)
        key = self.key_builder.build(filename=filename, directory=directory)
        if self.key_prefix:
            key = posixpath.join(self.key_prefix, key)
        await asyncio.to_thread(
            self._create_client().put_object,
            Bucket=self.bucket_name,
            Key=key,
            Body=content,
            **({"ContentType": file.content_type} if file.content_type else {}),
        )
        media = Media(
            object_key=key,
            bucket=self.bucket_name,
            original_filename=filename,
            content_type=file.content_type,
            media_type=self._media_type(file.content_type),
            size=len(content),
            checksum=hashlib.sha256(content).hexdigest(),
            uploaded_by=uploaded_by,
        )
        self.session.add(media)
        await self.session.flush()
        return StoredMedia(media.id, filename, file.content_type, len(content))

    async def get(self, media_id: int) -> tuple[StoredMedia, bytes] | None:
        """Читает медиа из MinIO по идентификатору записи."""
        media = await self.session.scalar(select(Media).where(Media.id == media_id, Media.deleted_at.is_(None)))
        if media is None:
            return None
        response = await asyncio.to_thread(
            self._create_client().get_object,
            Bucket=self.bucket_name,
            Key=media.object_key,
        )
        content = await asyncio.to_thread(response["Body"].read)
        return StoredMedia(media.id, media.original_filename, media.content_type, media.size), content

    async def get_default_avatar(self) -> tuple[StoredMedia, bytes] | None:
        """Читает системный дефолтный аватар по стабильному object key."""
        media = await self.session.scalar(
            select(Media).where(Media.object_key == DEFAULT_AVATAR_OBJECT_KEY, Media.deleted_at.is_(None))
        )
        if media is None:
            return None
        response = await asyncio.to_thread(
            self._create_client().get_object,
            Bucket=self.bucket_name,
            Key=media.object_key,
        )
        content = await asyncio.to_thread(response["Body"].read)
        return StoredMedia(media.id, media.original_filename, media.content_type, media.size), content

    async def delete(self, media_id: int) -> bool:
        """Удаляет объект из MinIO и помечает его запись удалённой."""
        media = await self.session.scalar(select(Media).where(Media.id == media_id, Media.deleted_at.is_(None)))
        if media is None:
            return False
        await asyncio.to_thread(
            self._create_client().delete_object,
            Bucket=self.bucket_name,
            Key=media.object_key,
        )
        from datetime import UTC, datetime

        media.deleted_at = datetime.now(UTC)
        await self.session.flush()
        return True

    @staticmethod
    def _media_type(content_type: str | None) -> str:
        """Определяет укрупнённый тип файла по MIME-типу."""
        if content_type:
            prefix = content_type.partition("/")[0]
            if prefix in {"image", "video", "audio"}:
                return prefix
        return "document"

    def _create_client(self) -> Any:
        """Создаёт boto3-клиент с параметрами MinIO/S3."""
        boto3 = importlib.import_module("boto3")
        kwargs: dict[str, Any] = {"service_name": "s3"}
        if self.region_name:
            kwargs["region_name"] = self.region_name
        if self.endpoint_url:
            kwargs["endpoint_url"] = self.endpoint_url
        if self.access_key_id:
            kwargs["aws_access_key_id"] = self.access_key_id.get_secret_value()
        if self.secret_access_key:
            kwargs["aws_secret_access_key"] = self.secret_access_key.get_secret_value()
        return boto3.client(**kwargs)
