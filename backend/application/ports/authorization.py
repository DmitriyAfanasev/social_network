from abc import ABC, abstractmethod


class AuthorizationService(ABC):
    @abstractmethod
    async def has_permission(self, user_id: int, permission: str) -> bool:
        pass
