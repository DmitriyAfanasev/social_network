from typing import ClassVar

import bcrypt


class ErrorMessages:
    EMPTY_PASSWORD = "Пароль не может быть пустым"
    PASSWORD_TOO_LONG = "Пароль слишком длинный (максимум {max_length} символов)"
    INVALID_CREDENTIALS = "Неверные учетные данные"
    PASSWORD_TYPE_ERROR = "Пароль должен быть строкой или bytes"
    HASH_TYPE_ERROR = "Хэш должен быть строкой или bytes"


class PasswordVerificationError(Exception):
    def __init__(
        self,
        message: str = ErrorMessages.INVALID_CREDENTIALS,
        status_code: int = 400,
    ) -> None:
        self.message = message
        self.status_code = status_code
        super().__init__(message)


class BcryptPasswordContext:
    def hash(self, password: str | bytes) -> str:
        password_bytes = self._to_bytes(password)
        return bcrypt.hashpw(password_bytes, bcrypt.gensalt()).decode("utf-8")

    def verify(
        self,
        plain_password: str | bytes,
        hashed_password: str | bytes,
    ) -> bool:
        return bcrypt.checkpw(
            self._to_bytes(plain_password),
            self._to_bytes(hashed_password),
        )

    def needs_update(self, hashed_password: str | bytes) -> bool:
        self._to_bytes(hashed_password)
        return False

    @staticmethod
    def _to_bytes(value: str | bytes) -> bytes:
        if isinstance(value, bytes):
            return value
        return value.encode("utf-8")


class PasswordHelper:
    MAX_PASSWORD_LENGTH = 72
    PWD_CONTEXT: ClassVar[BcryptPasswordContext] = BcryptPasswordContext()

    @classmethod
    def validate_password(cls, password: str) -> None:
        if not password:
            raise ValueError(ErrorMessages.EMPTY_PASSWORD)
        if len(password) > cls.MAX_PASSWORD_LENGTH:
            raise ValueError(
                ErrorMessages.PASSWORD_TOO_LONG.format(
                    max_length=cls.MAX_PASSWORD_LENGTH,
                )
            )

    @classmethod
    def generate_password(cls, password: str) -> str:
        cls.validate_password(password)
        return cls.PWD_CONTEXT.hash(password)

    @classmethod
    def verify_password(
        cls,
        plain_password: str | bytes,
        hashed_password: str | bytes,
    ) -> bool:
        if not isinstance(plain_password, (str, bytes)):
            raise TypeError(ErrorMessages.PASSWORD_TYPE_ERROR)
        if not isinstance(hashed_password, (str, bytes)):
            raise TypeError(ErrorMessages.HASH_TYPE_ERROR)

        try:
            return bool(cls.PWD_CONTEXT.verify(plain_password, hashed_password))
        except ValueError:
            return False
