from backend.application.ports.file_upload_service import FileUploadService, UploadFileSource
from backend.application.ports.media_storage import MediaStorage


class MediaFileUploadAdapter:
    """Legacy file-upload port backed by the metadata-aware media storage."""

    def __init__(self, storage: MediaStorage) -> None:
        self.storage = storage

    async def upload(self, *, file: UploadFileSource, directory: str) -> str:
        media = await self.storage.upload(file=file, directory=directory)
        return f"/media/{media.id}"

    async def delete(self, *, file_url: str) -> bool:
        media_id = file_url.removeprefix("/media/")
        if not media_id.isdigit():
            return False
        return await self.storage.delete(int(media_id))


def create_file_upload_service(
    media_storage: MediaStorage,
) -> FileUploadService:
    return MediaFileUploadAdapter(media_storage)
