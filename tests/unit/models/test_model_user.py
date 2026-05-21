from datetime import date

import pytest
from sqlalchemy import select
from sqlalchemy.exc import IntegrityError
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from backend.infra.models.sqlalchemy import Profile, User


@pytest.mark.asyncio(loop_scope="class")
class TestUserModel:
    """Группа тестов для модели пользователя User."""

    async def test_create_user(self, db_session: AsyncSession) -> None:
        """Тестирование корректного создания пользователя."""
        new_user = User(
            username="test_user",
            hashed_password="hashed_password_123",
            email="test@example.com",
        )
        db_session.add(new_user)
        await db_session.commit()

        stmt = (
            select(User)
            .where(User.username == "test_user")
            .options(selectinload(User.posts))
        )
        result = await db_session.execute(stmt)
        user = result.scalars().first()

        assert user is not None
        assert user.email == "test@example.com"
        assert user.is_active is True
        assert user.created_at is not None
        assert user.updated_at is not None
        assert user.posts == []
        assert user.is_superuser is False
        assert user.profile is None
        assert user.likes == []

        await db_session.delete(user)
        await db_session.commit()
        await db_session.close()

    async def test_delete_user(self, db_session: AsyncSession) -> None:
        """Тестирования удаления пользователя."""
        new_deleted_user = User(
            username="test_user_deleted",
            hashed_password="hashed_password_123",
            email="test_user_deleted@example.com",
        )
        db_session.add(new_deleted_user)
        await db_session.commit()

        stmt = await db_session.execute(
            select(User).where(User.username == "test_user_deleted")
        )
        last_user = stmt.scalars().first()
        assert last_user is not None

        await db_session.delete(last_user)
        await db_session.commit()

        result = await db_session.execute(
            select(User).where(User.username == "test_user_deleted")
        )
        none_user = result.scalars().first()
        assert none_user is None

    async def test_duplicate_username_raises_integrity_error(
        self, db_session: AsyncSession
    ) -> None:
        """Проверяет, что дублирование username вызывает ошибку целостности."""
        user1 = User(
            username="unique_user",
            hashed_password="pass1234",
            email="unique1@test.com",
        )
        db_session.add(user1)
        await db_session.commit()

        user2 = User(
            username="unique_user",
            hashed_password="pass1234",
            email="unique2@test.com",
        )
        db_session.add(user2)

        with pytest.raises(IntegrityError) as exc_info:
            await db_session.commit()

        assert "duplicate" in str(exc_info.value).lower()
        await db_session.rollback()
        await db_session.delete(user1)
        await db_session.commit()

    async def test_duplicate_email_raises_integrity_error(
        self, db_session: AsyncSession
    ) -> None:
        """Проверяет, что дублирование email вызывает ошибку целостности."""
        user1 = User(
            username="user1",
            hashed_password="pass1234",
            email="same_email@test.com",
        )
        db_session.add(user1)
        await db_session.commit()

        user2 = User(
            username="user2",
            hashed_password="pass1234",
            email="same_email@test.com",
        )
        db_session.add(user2)

        with pytest.raises(IntegrityError) as exc_info:
            await db_session.commit()

        assert "duplicate" in str(exc_info.value).lower()
        await db_session.rollback()

    @pytest.mark.parametrize(
        "username, password, email",
        [
            ("", "pass1234", "test@example.com"),
            ("user", "", "test@example.com"),
            ("user", "pass1234", ""),
        ],
    )
    async def test_empty_filed_raises_integrity_error(
        self,
        db_session: AsyncSession,
        username: str,
        password: str,
        email: str,
    ) -> None:
        """Проверяет, что пустое поле вызывает ошибку."""
        user = User(username=username, hashed_password=password, email=email)
        db_session.add(user)

        with pytest.raises(IntegrityError) as exc_info:
            await db_session.commit()

        assert "asyncpg.exceptions.CheckViolationError" in str(exc_info.value)
        await db_session.rollback()

    async def test_missing_required_fields_raises_integrity_error(
        self, db_session: AsyncSession
    ) -> None:
        """Проверяет, что отсутствие required полей вызывает ошибку."""
        user = User(hashed_password="pass1234")
        db_session.add(user)

        with pytest.raises(IntegrityError) as exc_info:
            await db_session.commit()

        assert "null value in column" in str(exc_info.value)
        await db_session.rollback()

    async def test_update_user_data(self, db_session: AsyncSession) -> None:
        """Тест обновления данных пользователя."""
        user = User(
            username="to_update",
            hashed_password="old_pass",
            email="update@test.com",
        )
        db_session.add(user)
        await db_session.commit()

        updated_user = await db_session.get(User, user.id)
        updated_user.hashed_password = "new_hashed_pass"
        await db_session.commit()

        assert updated_user is not None
        assert updated_user.hashed_password == "new_hashed_pass"

        await db_session.delete(updated_user)
        await db_session.commit()

    async def test_user_is_not_active_by_default(
        self, db_session: AsyncSession
    ) -> None:
        """Проверяет, что пользователь не активен, если явно не указано
        иное."""
        user = User(
            username="inactive_user",
            hashed_password="pass1234",
            email="inactive@test.com",
            is_active=False,
        )
        db_session.add(user)
        await db_session.commit()

        fetched_user = await db_session.get(User, user.id)
        assert fetched_user is not None
        assert fetched_user.is_active is False

        await db_session.delete(fetched_user)
        await db_session.commit()

    async def test_query_nonexistent_user_returns_none(
        self, db_session: AsyncSession
    ) -> None:
        """Проверяет, что запрос несуществующего пользователя возвращает
        None."""
        result = await db_session.execute(select(User).where(User.id == 9999))
        user = result.scalars().first()
        assert user is None

    async def test_set_profile_for_user(self, db_session: AsyncSession) -> None:
        user = User(
            username="profile_user",
            hashed_password="pass1234",
            email="profile_user@example.com",
        )
        db_session.add(user)
        await db_session.flush()
        profile = Profile(
            first_name="John",
            last_name="Doe",
            birth_date=date(2000, 1, 1),
            gender="Мужчина",
            phone_number="+89991055449",
            country="Россия",
            city="Питер",
            street="Мира 1",
            bio="Живу в Питере, и я из России.",
            user_id=user.id,
        )
        user.profile = profile
        db_session.add_all([user, profile])
        await db_session.commit()

        fetched_user = await db_session.get(User, user.id)
        assert fetched_user.profile is not None
        assert fetched_user.profile.first_name == "John"

        await db_session.delete(profile)
        await db_session.delete(fetched_user)
        await db_session.commit()

    async def test_error_deleted_user_without_deleted_profile(
        self, db_session: AsyncSession
    ) -> None:

        user = User(
            username="profile_user",
            hashed_password="pass1234",
            email="profile_user@example.com",
        )
        db_session.add(user)
        await db_session.flush()
        profile = Profile(
            first_name="John",
            last_name="Doe",
            birth_date=date(2000, 1, 1),
            gender="Мужчина",
            phone_number="+89991055449",
            country="Россия",
            city="Питер",
            street="Мира 1",
            bio="Живу в Питере, и я из России.",
            user_id=user.id,
        )
        user.profile = profile
        db_session.add_all([user, profile])
        await db_session.commit()

        fetched_user = await db_session.get(User, user.id)

        with pytest.raises(IntegrityError) as exc_info:
            await db_session.delete(fetched_user)
            await db_session.commit()

        assert "postgresql.asyncpg.integrityerror" in str(exc_info.value).lower()

        await db_session.rollback()
        await db_session.delete(profile)
        await db_session.delete(fetched_user)
        await db_session.commit()
