from typing import Protocol


class UploadFileSource(Protocol):
    filename: str | None
    size: int | None
    content_type: str | None

    async def read(self, size: int = -1) -> bytes:
        pass


class FileUploadService(Protocol):
    async def upload(
        self,
        *,
        file: UploadFileSource,
        directory: str,
    ) -> str:
        pass

    async def delete(self, *, file_url: str) -> bool:
        pass
