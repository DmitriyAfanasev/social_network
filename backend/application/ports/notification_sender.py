from abc import ABC, abstractmethod


class NotificationSender(ABC):
    @abstractmethod
    async def send_registration_confirmation(
        self,
        email: str,
        token: str,
        base_url: str,
    ) -> None:
        pass

    @abstractmethod
    async def send_password_reset(
        self,
        email: str,
        token: str,
        base_url: str,
    ) -> None:
        pass
