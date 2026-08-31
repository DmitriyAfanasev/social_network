from abc import ABC, abstractmethod
from datetime import datetime
from typing import Any


class MessageRepository(ABC):
    @abstractmethod
    async def get_or_create_direct(self, user_id: int, other_user_id: int) -> Any:
        pass

    @abstractmethod
    async def list_conversations(self, user_id: int) -> list[Any]:
        pass

    @abstractmethod
    async def list_messages(
        self,
        conversation_id: int,
        user_id: int,
        before: tuple[datetime, int] | None,
        limit: int,
    ) -> tuple[list[Any], bool]:
        pass

    @abstractmethod
    async def create_message(self, conversation_id: int, sender_id: int, text: str) -> Any:
        pass

    @abstractmethod
    async def mark_read(self, conversation_id: int, user_id: int, message_id: int) -> Any:
        pass
