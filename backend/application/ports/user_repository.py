from abc import ABC, abstractmethod

from backend.domain.user.entity import User


class UserRepository(ABC):
    @abstractmethod
    async def get_by_id(self, user_id: int) -> User | None:
        """
        Возвращает пользователя по его ID.

        :param user_id: ID пользователя.
        :return: Объект пользователя или None, если пользователь не найден.
        """
        pass

    @abstractmethod
    async def get_by_username(self, username: str) -> User | None:
        """
        Возвращает пользователя по его имени пользователя.

        :param username: Имя пользователя.
        :return: Объект пользователя или None, если пользователь не найден.
        """
        pass

    @abstractmethod
    async def get_by_email(self, email: str) -> User | None:
        """
        Возвращает пользователя по его email.

        :param email: Email пользователя.
        :return: Объект пользователя или None, если пользователь не найден.
        """
        pass

    @abstractmethod
    async def create(self, user: User) -> User:
        """
        Создает нового пользователя.

        :param user: Объект пользователя для создания.
        :return: Созданный объект пользователя.
        """
        pass

    @abstractmethod
    async def activate_by_email(self, email: str) -> User | None:
        pass

    @abstractmethod
    async def change_password_by_email(
        self,
        email: str,
        hashed_password: str,
    ) -> User | None:
        pass
