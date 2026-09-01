from abc import ABC, abstractmethod
from collections.abc import Sequence
from typing import Any


class AuditRepository(ABC):
    @abstractmethod
    async def record(
        self,
        actor_id: int,
        action: str,
        target_type: str,
        target_id: int | None,
        details: dict[str, Any] | None = None,
    ) -> None:
        pass

    @abstractmethod
    async def list(self, limit: int) -> Sequence[Any]:
        pass
