from abc import ABC, abstractmethod


class PendingTokenStore(ABC):
    @abstractmethod
    async def save_registration_confirmation_token(
        self,
        token: str,
        email: str,
        expires_sec: int = 1800,
    ) -> bool:
        pass

    @abstractmethod
    async def save_password_reset_token(
        self,
        token: str,
        email: str,
        expires_sec: int = 600,
    ) -> bool:
        pass

    @abstractmethod
    async def get_email_by_token(self, token: str) -> str | None:
        pass

    @abstractmethod
    async def delete_token(self, token: str) -> None:
        pass
