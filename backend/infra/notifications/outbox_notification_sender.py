from backend.application.event_types import (
    PASSWORD_RESET_REQUESTED_EVENT,
    REGISTRATION_CONFIRMATION_REQUESTED_EVENT,
)
from backend.application.events import IntegrationEvent
from backend.application.ports.notification_sender import NotificationSender
from backend.application.ports.outbox_repository import OutboxRepository
from backend.application.ports.transaction_manager import TransactionManager


class OutboxNotificationSender(NotificationSender):
    def __init__(
        self,
        outbox_repository: OutboxRepository,
        transaction_manager: TransactionManager,
    ) -> None:
        self.outbox_repository = outbox_repository
        self.transaction_manager = transaction_manager

    async def send_registration_confirmation(
        self,
        email: str,
        token: str,
        base_url: str,
    ) -> None:
        async with self.transaction_manager:
            await self.outbox_repository.add(
                IntegrationEvent(
                    event_type=REGISTRATION_CONFIRMATION_REQUESTED_EVENT,
                    payload={
                        "email": email,
                        "token": token,
                        "base_url": base_url,
                    },
                )
            )

    async def send_password_reset(
        self,
        email: str,
        token: str,
        base_url: str,
    ) -> None:
        async with self.transaction_manager:
            await self.outbox_repository.add(
                IntegrationEvent(
                    event_type=PASSWORD_RESET_REQUESTED_EVENT,
                    payload={
                        "email": email,
                        "token": token,
                        "base_url": base_url,
                    },
                )
            )
