from abc import ABC, abstractmethod
from datetime import datetime

from backend.application.events import IntegrationEvent
from backend.application.read_models import OutboxEventReadModel


class OutboxRepository(ABC):
    @abstractmethod
    async def add(self, event: IntegrationEvent) -> None:
        pass

    @abstractmethod
    async def get_pending(self, *, limit: int, now: datetime) -> list[OutboxEventReadModel]:
        pass

    @abstractmethod
    async def mark_published(self, event_id: int, *, published_at: datetime) -> None:
        pass

    @abstractmethod
    async def mark_failed(
        self,
        event_id: int,
        *,
        next_retry_at: datetime,
        last_error: str,
    ) -> None:
        pass
