from abc import ABC, abstractmethod
from typing import Any


class MessageRepository(ABC):
    @abstractmethod
    async def get_or_create_direct(self, user_id: int, other_user_id: int) -> Any:
        pass

    @abstractmethod
    async def list_conversations(self, user_id: int) -> list[Any]:
        pass

    @abstractmethod
    async def get_participant_ids(self, conversation_id: int) -> list[int]:
        pass

    @abstractmethod
    async def list_messages(
        self,
        conversation_id: int,
        user_id: int,
        offset: int,
        limit: int,
    ) -> tuple[list[Any], bool]:
        """Return one cursor-based page and whether another page exists."""
        pass

    @abstractmethod
    async def create_message(
        self, conversation_id: int, sender_id: int, text: str, media_id: int | None = None
    ) -> Any:
        pass

    @abstractmethod
    async def update_message(self, conversation_id: int, message_id: int, sender_id: int, text: str) -> Any:
        pass

    @abstractmethod
    async def remove_message_media(self, conversation_id: int, message_id: int, sender_id: int) -> tuple[Any, int | None]:
        pass

    @abstractmethod
    async def delete_message(self, conversation_id: int, message_id: int, sender_id: int) -> Any:
        pass

    @abstractmethod
    async def mark_read(self, conversation_id: int, user_id: int, message_id: int) -> Any:
        pass
