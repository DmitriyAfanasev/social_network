from abc import ABC, abstractmethod
from collections.abc import Sequence

from backend.application.read_models import ConversationReadModel, MessageReadModel


class MessageRepository(ABC):
    @abstractmethod
    async def get_or_create_direct(self, user_id: int, other_user_id: int) -> ConversationReadModel:
        pass

    @abstractmethod
    async def list_conversations(self, user_id: int, archived: bool = False) -> Sequence[ConversationReadModel]:
        pass

    @abstractmethod
    async def archive_conversation(self, conversation_id: int, user_id: int) -> None:
        pass

    @abstractmethod
    async def unarchive_conversation(self, conversation_id: int, user_id: int) -> None:
        pass

    @abstractmethod
    async def hide_conversation(self, conversation_id: int, user_id: int) -> None:
        pass

    @abstractmethod
    async def mark_unread(self, conversation_id: int, user_id: int) -> None:
        pass

    @abstractmethod
    async def pin_conversation(self, conversation_id: int, user_id: int, pinned: bool) -> None:
        pass

    @abstractmethod
    async def mute_conversation(self, conversation_id: int, user_id: int, muted: bool) -> None:
        pass

    @abstractmethod
    async def clear_history(self, conversation_id: int, user_id: int) -> None:
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
    ) -> tuple[Sequence[MessageReadModel], bool]:
        """Return one cursor-based page and whether another page exists."""
        pass

    @abstractmethod
    async def create_message(
        self, conversation_id: int, sender_id: int, text: str, media_id: int | None = None
    ) -> MessageReadModel:
        pass

    @abstractmethod
    async def update_message(self, conversation_id: int, message_id: int, sender_id: int, text: str) -> MessageReadModel:
        pass

    @abstractmethod
    async def remove_message_media(self, conversation_id: int, message_id: int, sender_id: int) -> tuple[MessageReadModel, int | None]:
        pass

    @abstractmethod
    async def delete_message(self, conversation_id: int, message_id: int, sender_id: int) -> MessageReadModel:
        pass

    @abstractmethod
    async def mark_read(self, conversation_id: int, user_id: int, message_id: int) -> bool:
        pass
