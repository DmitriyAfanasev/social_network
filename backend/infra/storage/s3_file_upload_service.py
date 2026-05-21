import asyncio
import importlib
import posixpath
from typing import Any
from urllib.parse import urlparse

from pydantic import SecretStr

from backend.application.ports.file_upload_service import FileUploadService, UploadFileSource
from backend.infra.storage.file_upload import (
    FileKeyBuilder,
    FileUploadSettings,
    FileUploadValidator,
    read_upload_file,
)


class S3FileUploadService(FileUploadService):
    def __init__(
        self,
        *,
        bucket_name: str,
        max_size_bytes: int,
        region_name: str | None = None,
        endpoint_url: str | None = None,
        access_key_id: SecretStr | None = None,
        secret_access_key: SecretStr | None = None,
        public_base_url: str | None = None,
        key_prefix: str = "",
    ) -> None:
        if not bucket_name:
            raise ValueError("S3 bucket_name is required.")

        self.bucket_name = bucket_name
        self.region_name = region_name
        self.endpoint_url = endpoint_url
        self.access_key_id = access_key_id
        self.secret_access_key = secret_access_key
        self.public_base_url = public_base_url.rstrip("/") if public_base_url else None
        self.key_prefix = key_prefix.strip("/")
        self.validator = FileUploadValidator(FileUploadSettings(max_size_bytes=max_size_bytes))
        self.key_builder = FileKeyBuilder()

    async def upload(
        self,
        *,
        file: UploadFileSource,
        directory: str,
    ) -> str:
        filename = self.validator.get_safe_filename(file.filename)
        storage_key = self.key_builder.build(
            filename=filename,
            directory=directory,
        )
        if self.key_prefix:
            storage_key = posixpath.join(self.key_prefix, storage_key)

        content = await read_upload_file(file, self.validator)
        client = self._create_client()
        put_kwargs: dict[str, Any] = {
            "Bucket": self.bucket_name,
            "Key": storage_key,
            "Body": content,
        }
        if file.content_type:
            put_kwargs["ContentType"] = file.content_type

        await asyncio.to_thread(client.put_object, **put_kwargs)
        return self._build_public_url(storage_key)

    def _create_client(self) -> Any:
        try:
            boto3 = importlib.import_module("boto3")
        except ModuleNotFoundError as e:
            raise RuntimeError("Для S3-хранилища установите зависимость boto3.") from e

        kwargs: dict[str, Any] = {"service_name": "s3"}
        if self.region_name is not None:
            kwargs["region_name"] = self.region_name
        if self.endpoint_url is not None:
            kwargs["endpoint_url"] = self.endpoint_url
        if self.access_key_id is not None:
            kwargs["aws_access_key_id"] = self.access_key_id.get_secret_value()
        if self.secret_access_key is not None:
            kwargs["aws_secret_access_key"] = self.secret_access_key.get_secret_value()

        return boto3.client(**kwargs)

    def _build_public_url(self, storage_key: str) -> str:
        if self.public_base_url:
            return f"{self.public_base_url}/{storage_key}"
        if self.region_name:
            return f"https://{self.bucket_name}.s3.{self.region_name}.amazonaws.com/{storage_key}"
        return f"https://{self.bucket_name}.s3.amazonaws.com/{storage_key}"

    async def delete(self, *, file_url: str) -> bool:
        storage_key = self._storage_key_from_url(file_url)
        if storage_key is None:
            return False

        client = self._create_client()
        await asyncio.to_thread(
            client.delete_object,
            Bucket=self.bucket_name,
            Key=storage_key,
        )
        return True

    def _storage_key_from_url(self, file_url: str) -> str | None:
        if self.public_base_url:
            public_prefix = f"{self.public_base_url}/"
            if not file_url.startswith(public_prefix):
                return None
            return file_url.removeprefix(public_prefix).strip("/") or None

        parsed_url = urlparse(file_url)
        storage_key = parsed_url.path.strip("/")
        return storage_key or None
