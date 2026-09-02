from abc import ABC, abstractmethod

from backend.domain.call import CallSession


class CallSessionStore(ABC):
    @abstractmethod
    async def save(self, session: CallSession) -> None:
        pass

    @abstractmethod
    async def get(self, call_id: str) -> CallSession | None:
        pass

    @abstractmethod
    async def delete(self, call_id: str) -> None:
        pass
