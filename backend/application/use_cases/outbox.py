from datetime import UTC, datetime, timedelta

from backend.application.ports.event_publisher import EventPublisher
from backend.application.ports.outbox_repository import OutboxRepository
from backend.application.ports.transaction_manager import TransactionManager


class PublishOutboxEventsUseCase:
    """Публикует накопленные outbox-события во внешний event bus.

    Use case читает pending/failed события из БД, пробует отправить каждое через
    `EventPublisher`, а затем фиксирует результат в outbox-таблице. При ошибке
    событие не теряется: оно получает `failed`, `attempts + 1` и `next_retry_at`.
    """

    def __init__(
        self,
        outbox_repository: OutboxRepository,
        event_publisher: EventPublisher,
        transaction_manager: TransactionManager,
    ) -> None:
        self.outbox_repository = outbox_repository
        self.event_publisher = event_publisher
        self.transaction_manager = transaction_manager

    async def execute(self, *, limit: int) -> int:
        """Опубликовать до `limit` событий и вернуть количество успешных publish."""
        now = datetime.now(UTC)
        events = await self.outbox_repository.get_pending(limit=limit, now=now)
        published_count = 0

        for event in events:
            try:
                await self.event_publisher.publish(
                    event_type=event.event_type,
                    payload=event.payload,
                )
            except (OSError, RuntimeError, ValueError) as exc:
                retry_delay = self._retry_delay(event.attempts + 1)
                async with self.transaction_manager:
                    await self.outbox_repository.mark_failed(
                        event.id,
                        next_retry_at=datetime.now(UTC) + retry_delay,
                        last_error=str(exc),
                    )
                continue

            async with self.transaction_manager:
                await self.outbox_repository.mark_published(
                    event.id,
                    published_at=datetime.now(UTC),
                )
            published_count += 1

        return published_count

    @classmethod
    def _retry_delay(cls, attempts: int) -> timedelta:
        """Посчитать задержку следующей попытки с простым exponential backoff."""
        seconds = min(300, 2**attempts)
        return timedelta(seconds=seconds)
