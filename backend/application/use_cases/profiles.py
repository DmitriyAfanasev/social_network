from typing import cast

from fastapi import UploadFile

from backend.application.analytics_events import build_analytics_event_payload
from backend.application.commands import ProfileUpdateCommand
from backend.application.event_types import PROFILE_PHOTO_DELETED_EVENT
from backend.application.events import IntegrationEvent
from backend.application.exceptions import (
    PermissionDeniedError,
    ValidationAppError,
)
from backend.application.ports.file_upload_service import FileUploadService
from backend.application.ports.friend_repository import FriendRepository
from backend.application.ports.outbox_repository import OutboxRepository
from backend.application.ports.post_repository import PostRepository
from backend.application.ports.profile_repository import ProfileRepository
from backend.application.ports.transaction_manager import TransactionManager
from backend.application.read_models import ProfilePhotoAlbumReadModel
from backend.application.results import (
    AvatarHistoryResult,
    AvatarRemoveResult,
    AvatarUploadResult,
    MessageResult,
    PhotoAlbumResult,
    ProfilePhotoAlbumResult,
    ProfilePhotoResult,
    ProfilePhotosResult,
    ProfileResult,
    UserResult,
)
from backend.domain.user.entity import User
from backend.infra.config import DEFAULT_PATH_TO_AVATAR


class GetProfileUseCase:
    def __init__(
        self,
        profile_repository: ProfileRepository,
        post_repository: PostRepository,
        friend_repository: FriendRepository,
    ) -> None:
        self.profile_repository = profile_repository
        self.post_repository = post_repository
        self.friend_repository = friend_repository

    async def execute(self, current_user: User, profile_id: int) -> ProfileResult:
        current_user_id = cast(int, current_user.id)
        profile_user = await self.profile_repository.get_by_id(profile_id)
        is_own_profile = current_user_id == profile_user.id
        is_friend = (
            False
            if is_own_profile
            else await self.friend_repository.is_friend(current_user_id, profile_id)
        )
        is_subscribed = (
            False
            if is_own_profile
            else await self.friend_repository.is_subscribed(current_user_id, profile_id)
        )
        is_subscribed_to_current = (
            False
            if is_own_profile
            else await self.friend_repository.is_subscribed(profile_id, current_user_id)
        )
        posts = await self.post_repository.get_all_by_author_id(
            profile_id,
            current_user_id,
        )

        return ProfileResult(
            user=profile_user,
            is_own_profile=is_own_profile,
            is_friend=is_friend,
            is_subscribed=is_subscribed,
            is_subscribed_to_current=is_subscribed_to_current,
            current_user=current_user,
            posts=posts,
        )


class UpdateProfileUseCase:
    def __init__(
        self,
        profile_repository: ProfileRepository,
        transaction_manager: TransactionManager,
    ) -> None:
        self.profile_repository = profile_repository
        self.transaction_manager = transaction_manager

    async def execute(
        self,
        current_user: User,
        profile_id: int,
        command: ProfileUpdateCommand,
    ) -> UserResult:
        current_user_id = cast(int, current_user.id)
        profile_user = await self.profile_repository.get_by_id(profile_id)
        if profile_user.id != current_user_id:
            raise PermissionDeniedError("Это профиль чужого пользователя.")
        async with self.transaction_manager:
            profile_user = await self.profile_repository.update(profile_id, command)
        return UserResult(user=profile_user)


class UploadAvatarUseCase:
    def __init__(
        self,
        profile_repository: ProfileRepository,
        transaction_manager: TransactionManager,
        file_upload_service: FileUploadService,
    ) -> None:
        self.profile_repository = profile_repository
        self.transaction_manager = transaction_manager
        self.file_upload_service = file_upload_service

    async def execute(self, current_user: User, avatar: UploadFile) -> AvatarUploadResult:
        current_user_id = cast(int, current_user.id)
        try:
            image_url = await self.file_upload_service.upload(
                file=avatar,
                directory=f"users/{current_user_id}/avatars",
            )
            async with self.transaction_manager:
                await self.profile_repository.update_avatar(
                    current_user_id,
                    image_url,
                )
                await self.profile_repository.add_avatar_to_history(
                    current_user_id,
                    image_url,
                )
            return AvatarUploadResult(
                message="Аватар успешно обновлен",
                avatar_url=image_url,
            )
        except ValueError as e:
            raise ValidationAppError(str(e)) from e
        except Exception as e:
            raise ValidationAppError(f"Ошибка при загрузке аватара: {e!s}") from e


class RemoveAvatarUseCase:
    def __init__(
        self,
        profile_repository: ProfileRepository,
        transaction_manager: TransactionManager,
    ) -> None:
        self.profile_repository = profile_repository
        self.transaction_manager = transaction_manager

    async def execute(self, current_user: User) -> AvatarRemoveResult:
        current_user_id = cast(int, current_user.id)
        async with self.transaction_manager:
            await self.profile_repository.delete_avatar(
                current_user_id,
                DEFAULT_PATH_TO_AVATAR,
            )
        return AvatarRemoveResult(
            new_avatar=DEFAULT_PATH_TO_AVATAR,
            message="Аватар удален",
        )


class GetAvatarHistoryUseCase:
    def __init__(self, profile_repository: ProfileRepository) -> None:
        self.profile_repository = profile_repository

    async def execute(self, current_user: User) -> AvatarHistoryResult:
        current_user_id = cast(int, current_user.id)
        profile_user = await self.profile_repository.get_by_id(current_user_id)
        current_avatar = profile_user.profile.avatar if profile_user.profile else None
        avatars = await self.profile_repository.get_avatar_history(current_user_id)
        return AvatarHistoryResult(current_avatar=current_avatar, avatars=avatars)


class SelectAvatarUseCase:
    def __init__(
        self,
        profile_repository: ProfileRepository,
        transaction_manager: TransactionManager,
    ) -> None:
        self.profile_repository = profile_repository
        self.transaction_manager = transaction_manager

    async def execute(self, current_user: User, avatar_url: str) -> AvatarUploadResult:
        current_user_id = cast(int, current_user.id)
        if not await self.profile_repository.avatar_exists_in_history(current_user_id, avatar_url):
            raise ValidationAppError("Аватар не найден в истории")

        async with self.transaction_manager:
            await self.profile_repository.update_avatar(current_user_id, avatar_url)

        return AvatarUploadResult(
            message="Аватар выбран",
            avatar_url=avatar_url,
        )


class GetProfilePhotosUseCase:
    def __init__(self, profile_repository: ProfileRepository) -> None:
        self.profile_repository = profile_repository

    async def execute(self, current_user: User, profile_id: int) -> ProfilePhotosResult:
        current_user_id = cast(int, current_user.id)
        profile_user = await self.profile_repository.get_by_id(profile_id)
        avatar_history = await self.profile_repository.get_avatar_history(profile_id)
        custom_albums = await self.profile_repository.get_photo_albums(profile_id)

        avatar_album = ProfilePhotoAlbumResult(
            id=None,
            title="Аватары профиля",
            kind="avatars",
            created_at=avatar_history[0].created_at if avatar_history else None,
            photos=[
                ProfilePhotoResult(
                    id=None,
                    album_id=None,
                    photo_url=avatar.avatar_url,
                    caption=None,
                    created_at=avatar.created_at,
                )
                for avatar in avatar_history
            ],
        )
        albums = [
            avatar_album,
            *[self._album_to_result(album) for album in custom_albums],
        ]
        return ProfilePhotosResult(
            user=profile_user,
            is_own_profile=current_user_id == profile_id,
            albums=albums,
        )

    @staticmethod
    def _album_to_result(album: ProfilePhotoAlbumReadModel) -> ProfilePhotoAlbumResult:
        return ProfilePhotoAlbumResult(
            id=album.id,
            title=album.title,
            kind="custom",
            created_at=album.created_at,
            photos=[
                ProfilePhotoResult(
                    id=photo.id,
                    album_id=photo.album_id,
                    photo_url=photo.photo_url,
                    caption=photo.caption,
                    created_at=photo.created_at,
                )
                for photo in album.photos
            ],
        )


class CreatePhotoAlbumUseCase:
    def __init__(
        self,
        profile_repository: ProfileRepository,
        transaction_manager: TransactionManager,
    ) -> None:
        self.profile_repository = profile_repository
        self.transaction_manager = transaction_manager

    async def execute(self, current_user: User, title: str) -> PhotoAlbumResult:
        current_user_id = cast(int, current_user.id)
        normalized_title = title.strip()
        if not normalized_title:
            raise ValidationAppError("Название альбома не может быть пустым")

        async with self.transaction_manager:
            album = await self.profile_repository.create_photo_album(
                current_user_id,
                normalized_title,
            )

        return PhotoAlbumResult(album=GetProfilePhotosUseCase._album_to_result(album))


class UploadProfilePhotoUseCase:
    def __init__(
        self,
        profile_repository: ProfileRepository,
        transaction_manager: TransactionManager,
        file_upload_service: FileUploadService,
    ) -> None:
        self.profile_repository = profile_repository
        self.transaction_manager = transaction_manager
        self.file_upload_service = file_upload_service

    async def execute(
        self,
        current_user: User,
        album_id: int,
        photo: UploadFile,
        caption: str | None = None,
    ) -> PhotoAlbumResult:
        current_user_id = cast(int, current_user.id)
        album = await self.profile_repository.get_photo_album(album_id)
        if album.user_id != current_user_id:
            raise PermissionDeniedError("Нельзя загружать фото в чужой альбом")

        try:
            photo_url = await self.file_upload_service.upload(
                file=photo,
                directory=f"users/{current_user_id}/albums/{album_id}",
            )
        except ValueError as e:
            raise ValidationAppError(str(e)) from e

        async with self.transaction_manager:
            updated_album = await self.profile_repository.add_photo_to_album(
                album_id=album_id,
                user_id=current_user_id,
                photo_url=photo_url,
                caption=caption.strip() if caption else None,
            )

        return PhotoAlbumResult(album=GetProfilePhotosUseCase._album_to_result(updated_album))


class DeleteProfilePhotoUseCase:
    def __init__(
        self,
        profile_repository: ProfileRepository,
        outbox_repository: OutboxRepository,
        transaction_manager: TransactionManager,
    ) -> None:
        self.profile_repository = profile_repository
        self.outbox_repository = outbox_repository
        self.transaction_manager = transaction_manager

    async def execute(self, current_user: User, photo_id: int) -> MessageResult:
        current_user_id = cast(int, current_user.id)
        async with self.transaction_manager:
            deleted_file_url = await self.profile_repository.delete_photo(
                user_id=current_user_id,
                photo_id=photo_id,
            )
            if deleted_file_url is None:
                raise ValidationAppError("Фотография не найдена или недоступна")

            await self.outbox_repository.add(
                IntegrationEvent(
                    event_type=PROFILE_PHOTO_DELETED_EVENT,
                    payload=build_analytics_event_payload(
                        user_id=current_user_id,
                        entity_type="profile_photo",
                        entity_id=photo_id,
                        data={
                            "photo_id": photo_id,
                            "file_url": deleted_file_url,
                        },
                    ),
                )
            )

        return MessageResult(message="Фотография удалена")
