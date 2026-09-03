from abc import ABC, abstractmethod
from collections.abc import Sequence
from typing import Any

from backend.domain.user.entity import User


class AdminRepository(ABC):
    @abstractmethod
    async def list_roles(self) -> Sequence[User]:
        pass

    @abstractmethod
    async def assign_role(self, user_id: int, role_name: str) -> None:
        pass

    @abstractmethod
    async def remove_role(self, user_id: int, role_name: str) -> bool:
        pass

    @abstractmethod
    async def list_audit_logs(self, limit: int) -> Sequence[Any]:
        pass
