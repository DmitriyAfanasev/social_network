from abc import ABC, abstractmethod

from backend.application.events import JsonPayload


class EventPublisher(ABC):
    @abstractmethod
    async def publish(self, *, event_type: str, payload: JsonPayload) -> None:
        pass
