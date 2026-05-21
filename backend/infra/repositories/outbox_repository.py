from datetime import UTC, datetime

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from backend.application.events import IntegrationEvent
from backend.application.ports.outbox_repository import OutboxRepository as OutboxRepositoryPort
from backend.application.read_models import OutboxEventReadModel
from backend.infra.models.sqlalchemy import OutboxEvent


class OutboxRepository(OutboxRepositoryPort):
    def __init__(self, session: AsyncSession) -> None:
        self.session = session

    async def add(self, event: IntegrationEvent) -> None:
        self.session.add(
            OutboxEvent(
                event_type=event.event_type,
                payload=event.payload,
                next_retry_at=datetime.now(UTC),
            )
        )
        await self.session.flush()

    async def get_pending(self, *, limit: int, now: datetime) -> list[OutboxEventReadModel]:
        statement = (
            select(OutboxEvent)
            .where(
                OutboxEvent.status.in_(("pending", "failed")),
                OutboxEvent.next_retry_at <= now,
            )
            .order_by(OutboxEvent.created_at, OutboxEvent.id)
            .limit(limit)
            .with_for_update(skip_locked=True)
        )
        result = await self.session.execute(statement)
        return list(result.scalars().all())

    async def mark_published(self, event_id: int, *, published_at: datetime) -> None:
        event = await self.session.get(OutboxEvent, event_id)
        if event is None:
            return

        event.status = "published"
        event.published_at = published_at
        event.last_error = None
        self.session.add(event)
        await self.session.flush()

    async def mark_failed(
        self,
        event_id: int,
        *,
        next_retry_at: datetime,
        last_error: str,
    ) -> None:
        event = await self.session.get(OutboxEvent, event_id)
        if event is None:
            return

        event.status = "failed"
        event.attempts += 1
        event.next_retry_at = next_retry_at
        event.last_error = last_error[:2000]
        self.session.add(event)
        await self.session.flush()
