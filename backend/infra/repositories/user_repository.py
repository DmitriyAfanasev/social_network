
from datetime import UTC, datetime

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from backend.application.ports.user_repository import UserRepository as UserPort
from backend.domain.user.entity import Profile, User
from backend.infra.models.sqlalchemy.profile import Profile as ProfileModel
from backend.infra.models.sqlalchemy.user import User as UserModel


class UserRepository(UserPort):
    def __init__(self, session: AsyncSession) -> None:
        self.session = session

    async def get_by_id(self, user_id: int) -> User | None:
        result = await self.session.execute(
            select(UserModel)
            .options(selectinload(UserModel.profile))
            .where(UserModel.id == user_id)
        )
        user_model = result.scalar_one_or_none()
        if user_model is None:
            return None
        return self._to_domain(user_model)

    async def get_by_username(self, username: str) -> User | None:
        result = await self.session.execute(
            select(UserModel)
            .options(selectinload(UserModel.profile))
            .where(UserModel.username == username)
        )
        user_model = result.scalar_one_or_none()
        if user_model is None:
            return None
        return self._to_domain(user_model)

    async def get_by_email(self, email: str) -> User | None:
        result = await self.session.execute(
            select(UserModel)
            .options(selectinload(UserModel.profile))
            .where(UserModel.email == email)
        )
        user_model = result.scalar_one_or_none()
        if user_model is None:
            return None
        return self._to_domain(user_model)

    async def touch_last_seen(self, user_id: int) -> None:
        user = await self.session.get(UserModel, user_id)
        if user is not None:
            user.last_seen_at = datetime.now(UTC)
            await self.session.commit()

    async def create(self, user: User) -> User:
        user_model = self._to_orm(user)
        self.session.add(user_model)
        await self.session.flush()
        await self.session.refresh(user_model)
        return self._to_domain(user_model)

    async def activate_by_email(self, email: str) -> User | None:
        result = await self.session.execute(
            select(UserModel)
            .options(selectinload(UserModel.profile))
            .where(UserModel.email == email)
        )
        user_model = result.scalar_one_or_none()
        if user_model is None:
            return None

        user_model.is_active = True
        await self.session.flush()
        await self.session.refresh(user_model)
        return self._to_domain(user_model)

    async def change_password_by_email(
        self,
        email: str,
        hashed_password: str,
    ) -> User | None:
        result = await self.session.execute(
            select(UserModel)
            .options(selectinload(UserModel.profile))
            .where(UserModel.email == email)
        )
        user_model = result.scalar_one_or_none()
        if user_model is None:
            return None

        user_model.hashed_password = hashed_password
        await self.session.flush()
        await self.session.refresh(user_model)
        return self._to_domain(user_model)

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
                avatar=user_model.profile.avatar,
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

    def _to_orm(self, user: User) -> UserModel:
        user_model = UserModel(
            username=user.username,
            email=user.email,
            hashed_password=user.hashed_password,
            is_active=user.is_active,
            is_superuser=user.is_superuser,
            created_at=user.created_at,
        )
        if user.profile:
            profile_model = ProfileModel(
                first_name=user.profile.first_name,
                last_name=user.profile.last_name,
                middle_name=user.profile.middle_name,
                birth_date=user.profile.birth_date,
                gender=user.profile.gender,
                phone_number=user.profile.phone_number,
                country=user.profile.country,
                city=user.profile.city,
                street=user.profile.street,
                bio=user.profile.bio,
                avatar=user.profile.avatar,
            )
            user_model.profile = profile_model
        return user_model
