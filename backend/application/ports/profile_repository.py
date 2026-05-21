from abc import ABC, abstractmethod

from backend.application.commands import ProfileUpdateCommand
from backend.application.read_models import ProfileAvatarReadModel, ProfilePhotoAlbumReadModel
from backend.domain.user.entity import User


class ProfileRepository(ABC):
    @abstractmethod
    async def get_by_id(self, user_id: int) -> User:
        pass

    @abstractmethod
    async def update(self, user_id: int, command: ProfileUpdateCommand) -> User:
        pass

    @abstractmethod
    async def update_avatar(self, user_id: int, avatar_url: str) -> None:
        pass

    @abstractmethod
    async def delete_avatar(self, user_id: int, default_avatar: str) -> str:
        pass

    @abstractmethod
    async def add_avatar_to_history(self, user_id: int, avatar_url: str) -> None:
        pass

    @abstractmethod
    async def get_avatar_history(self, user_id: int) -> list[ProfileAvatarReadModel]:
        pass

    @abstractmethod
    async def avatar_exists_in_history(self, user_id: int, avatar_url: str) -> bool:
        pass

    @abstractmethod
    async def get_photo_albums(self, user_id: int) -> list[ProfilePhotoAlbumReadModel]:
        pass

    @abstractmethod
    async def create_photo_album(self, user_id: int, title: str) -> ProfilePhotoAlbumReadModel:
        pass

    @abstractmethod
    async def get_photo_album(self, album_id: int) -> ProfilePhotoAlbumReadModel:
        pass

    @abstractmethod
    async def add_photo_to_album(
        self,
        *,
        album_id: int,
        user_id: int,
        photo_url: str,
        caption: str | None = None,
    ) -> ProfilePhotoAlbumReadModel:
        pass

    @abstractmethod
    async def delete_photo(self, *, user_id: int, photo_id: int) -> str | None:
        pass
