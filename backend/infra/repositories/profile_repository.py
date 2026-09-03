from typing import cast

from sqlalchemy import delete, select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from backend.application.commands import ProfileUpdateCommand
from backend.application.exceptions import NotFoundError
from backend.application.ports.profile_repository import ProfileRepository as ProfilePort
from backend.application.read_models import ProfileAvatarReadModel, ProfilePhotoAlbumReadModel
from backend.domain.user.entity import Profile, User
from backend.infra.models.sqlalchemy import Profile as ProfileModel
from backend.infra.models.sqlalchemy import ProfileAvatar, ProfilePhoto, ProfilePhotoAlbum
from backend.infra.models.sqlalchemy import User as UserModel


class ProfileRepository(ProfilePort):
    def __init__(self, session: AsyncSession) -> None:
        self.session = session

    async def get_by_id(self, user_id: int) -> User:
        user = await self._get_model_by_id(user_id)
        return self._to_domain(user)

    async def update(self, user_id: int, command: ProfileUpdateCommand) -> User:
        user = await self._get_model_by_id(user_id)
        profile_data = {
            key: value
            for key, value in command.__dict__.items()
            if value is not None
        }
        if user.profile is None:
            user.profile = ProfileModel(user_id=user.id, **profile_data)
        else:
            for key, value in profile_data.items():
                setattr(user.profile, key, value)

        self.session.add(user)
        await self.session.flush()
        return await self.get_by_id(user_id)

    async def update_avatar(self, user_id: int, avatar_url: str) -> None:
        user = await self._get_model_by_id(user_id)
        if user.profile is None:
            user.profile = ProfileModel(user_id=user.id, avatar=avatar_url)
        else:
            user.profile.avatar = avatar_url
        self.session.add(user)
        await self.session.flush()

    async def delete_avatar(self, user_id: int, default_avatar: str) -> str:
        await self.update_avatar(user_id=user_id, avatar_url=default_avatar)
        return default_avatar

    async def add_avatar_to_history(self, user_id: int, avatar_url: str) -> None:
        if await self.avatar_exists_in_history(user_id, avatar_url):
            return

        self.session.add(ProfileAvatar(user_id=user_id, avatar_url=avatar_url))
        await self.session.flush()

    async def get_avatar_history(self, user_id: int) -> list[ProfileAvatarReadModel]:
        statement = (
            select(ProfileAvatar)
            .where(ProfileAvatar.user_id == user_id)
            .order_by(ProfileAvatar.created_at.desc(), ProfileAvatar.id.desc())
        )
        result = await self.session.execute(statement)
        return list(result.scalars().all())

    async def avatar_exists_in_history(self, user_id: int, avatar_url: str) -> bool:
        statement = select(ProfileAvatar.id).where(
            ProfileAvatar.user_id == user_id,
            ProfileAvatar.avatar_url == avatar_url,
        )
        return await self.session.scalar(statement) is not None

    async def get_photo_albums(self, user_id: int) -> list[ProfilePhotoAlbumReadModel]:
        statement = (
            select(ProfilePhotoAlbum)
            .where(ProfilePhotoAlbum.user_id == user_id)
            .options(selectinload(ProfilePhotoAlbum.photos))
            .order_by(ProfilePhotoAlbum.created_at.desc(), ProfilePhotoAlbum.id.desc())
        )
        result = await self.session.execute(statement)
        return cast(list[ProfilePhotoAlbumReadModel], list(result.scalars().all()))

    async def create_photo_album(self, user_id: int, title: str) -> ProfilePhotoAlbumReadModel:
        album = ProfilePhotoAlbum(user_id=user_id, title=title)
        self.session.add(album)
        await self.session.flush()
        await self.session.refresh(album, attribute_names=["photos"])
        return cast(ProfilePhotoAlbumReadModel, album)

    async def get_photo_album(self, album_id: int) -> ProfilePhotoAlbumReadModel:
        result = await self.session.execute(
            select(ProfilePhotoAlbum)
            .options(selectinload(ProfilePhotoAlbum.photos))
            .where(ProfilePhotoAlbum.id == album_id)
        )
        album = result.scalar_one_or_none()
        if album is None:
            raise NotFoundError(f"Альбом с ID {album_id} не найден")
        return cast(ProfilePhotoAlbumReadModel, album)

    async def add_photo_to_album(
        self,
        *,
        album_id: int,
        user_id: int,
        photo_url: str,
        caption: str | None = None,
    ) -> ProfilePhotoAlbumReadModel:
        self.session.add(
            ProfilePhoto(
                album_id=album_id,
                user_id=user_id,
                photo_url=photo_url,
                caption=caption,
            )
        )
        await self.session.flush()
        return await self.get_photo_album(album_id)

    async def delete_photo(self, *, user_id: int, photo_id: int) -> str | None:
        photo_url = await self.session.scalar(
            select(ProfilePhoto.photo_url).where(
                ProfilePhoto.id == photo_id,
                ProfilePhoto.user_id == user_id,
            )
        )
        if photo_url is not None:
            await self.session.execute(
                delete(ProfilePhoto).where(
                    ProfilePhoto.id == photo_id,
                    ProfilePhoto.user_id == user_id,
                )
            )
            return photo_url

        avatar = await self.session.scalar(
            select(ProfileAvatar).where(
                ProfileAvatar.id == photo_id,
                ProfileAvatar.user_id == user_id,
            )
        )
        if avatar is None:
            return None

        current_avatar = await self.session.scalar(
            select(ProfileModel.avatar).where(ProfileModel.user_id == user_id)
        )
        if avatar.avatar_url == current_avatar:
            return None

        await self.session.delete(avatar)
        return avatar.avatar_url

    async def _get_model_by_id(self, user_id: int) -> UserModel:
        result = await self.session.execute(
            select(UserModel)
            .options(selectinload(UserModel.profile))
            .where(UserModel.id == user_id)
        )
        user = result.scalar_one_or_none()
        if user is None:
            raise NotFoundError(f"Пользователь с ID {user_id} не найден")
        return user

    def _to_domain(self, user_model: UserModel) -> User:
        profile = None
        if user_model.profile:
            profile = Profile(
                first_name=user_model.profile.first_name,
                last_name=user_model.profile.last_name,
                middle_name=user_model.profile.middle_name,
                birth_date=user_model.profile.birth_date,
                gender=user_model.profile.gender,
                phone_number=user_model.profile.phone_number,
                country=user_model.profile.country,
                city=user_model.profile.city,
                street=user_model.profile.street,
                bio=user_model.profile.bio,
                status=user_model.profile.status,
                avatar=user_model.profile.avatar,
                profile_visibility=user_model.profile.profile_visibility,
                friend_request_policy=user_model.profile.friend_request_policy,
                message_policy=user_model.profile.message_policy,
                show_email=user_model.profile.show_email,
                show_phone=user_model.profile.show_phone,
                show_birth_date=user_model.profile.show_birth_date,
                show_friends=user_model.profile.show_friends,
                show_posts=user_model.profile.show_posts,
            )
        return User(
            id=user_model.id,
            username=user_model.username,
            email=user_model.email,
            hashed_password=user_model.hashed_password,
            is_active=user_model.is_active,
            is_superuser=user_model.is_superuser,
            last_seen_at=user_model.last_seen_at,
            created_at=user_model.created_at,
            profile=profile,
        )
