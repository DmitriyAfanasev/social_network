import base64
import json
from datetime import datetime

from backend.application.analytics_events import build_analytics_event_payload
from backend.application.event_types import (
    MESSAGE_DELETED_EVENT,
    MESSAGE_EDITED_EVENT,
    MESSAGE_MEDIA_REMOVED_EVENT,
    MESSAGE_READ_EVENT,
    MESSAGE_SENT_EVENT,
)
from backend.application.events import IntegrationEvent
from backend.application.exceptions import NotFoundError, ValidationAppError
from backend.application.ports.media_storage import MediaStorage
from backend.application.ports.message_repository import MessageRepository
from backend.application.ports.outbox_repository import OutboxRepository
from backend.application.ports.transaction_manager import TransactionManager
from backend.application.results import (
    ConversationResult,
    MessageItemResult,
    MessagePageResult,
)


def encode_cursor(created_at: datetime, message_id: int) -> str:
    value = json.dumps({"created_at": created_at.isoformat(), "id": message_id}).encode()
    return base64.urlsafe_b64encode(value).decode().rstrip("=")


def decode_cursor(cursor: str | None) -> tuple[datetime, int] | None:
    if cursor is None:
        return None
    try:
        padded = cursor + "=" * (-len(cursor) % 4)
        value = json.loads(base64.urlsafe_b64decode(padded).decode())
        return datetime.fromisoformat(value["created_at"]), int(value["id"])
    except (ValueError, TypeError, KeyError, json.JSONDecodeError) as error:
        raise ValidationAppError("Некорректный курсор") from error


class MessagingUseCase:
    def __init__(
        self,
        repository: MessageRepository,
        outbox_repository: OutboxRepository,
        transaction_manager: TransactionManager,
        media_storage: MediaStorage | None = None,
    ) -> None:
        self.repository = repository
        self.outbox_repository = outbox_repository
        self.transaction_manager = transaction_manager
        self.media_storage = media_storage

    async def get_or_create_direct(self, user_id: int, other_user_id: int) -> ConversationResult:
        if user_id == other_user_id:
            raise ValidationAppError("Нельзя создать диалог с самим собой")
        async with self.transaction_manager:
            conversation = await self.repository.get_or_create_direct(user_id, other_user_id)
        return ConversationResult(conversation=conversation)

    async def list_conversations(self, user_id: int) -> list[ConversationResult]:
        return [ConversationResult(conversation=item) for item in await self.repository.list_conversations(user_id)]

    async def get_participant_ids(self, conversation_id: int) -> list[int]:
        return await self.repository.get_participant_ids(conversation_id)

    async def list_messages(
        self, conversation_id: int, user_id: int, offset: int, limit: int
    ) -> MessagePageResult:
        if offset < 0:
            raise ValidationAppError("Параметр offset не может быть отрицательным")
        if not 1 <= limit <= 100:
            raise ValidationAppError("Параметр limit должен быть от 1 до 100")
        messages, has_more = await self.repository.list_messages(
            conversation_id, user_id, offset, limit
        )
        next_cursor = encode_cursor(messages[-1].created_at, messages[-1].id) if has_more else None
        return MessagePageResult(messages=messages, next_cursor=next_cursor, has_more=has_more)

    async def send_message(
        self, conversation_id: int, user_id: int, text: str, media_id: int | None = None
    ) -> MessageItemResult:
        normalized = text.strip()
        if not normalized and media_id is None:
            raise ValidationAppError("Сообщение не может быть пустым")
        if len(normalized) > 5000:
            raise ValidationAppError("Сообщение слишком длинное")
        async with self.transaction_manager:
            message = await self.repository.create_message(conversation_id, user_id, normalized, media_id)
            await self._record_message_event(MESSAGE_SENT_EVENT, conversation_id, user_id, message.id)
        return MessageItemResult(message=message)

    async def edit_message(self, conversation_id: int, message_id: int, user_id: int, text: str) -> MessageItemResult:
        """Редактирует текст сообщения, если текущий пользователь является его автором."""
        normalized = text.strip()
        if not normalized:
            raise ValidationAppError("Сообщение не может быть пустым")
        if len(normalized) > 5000:
            raise ValidationAppError("Сообщение слишком длинное")
        async with self.transaction_manager:
            message = await self.repository.update_message(conversation_id, message_id, user_id, normalized)
            await self._record_message_event(MESSAGE_EDITED_EVENT, conversation_id, user_id, message_id)
        return MessageItemResult(message=message)

    async def remove_message_media(self, conversation_id: int, message_id: int, user_id: int) -> MessageItemResult:
        async with self.transaction_manager:
            message, media_id = await self.repository.remove_message_media(conversation_id, message_id, user_id)
            if media_id is not None and self.media_storage is not None:
                await self.media_storage.delete(media_id)
            await self._record_message_event(MESSAGE_MEDIA_REMOVED_EVENT, conversation_id, user_id, message_id)
        return MessageItemResult(message=message)

    async def delete_message(self, conversation_id: int, message_id: int, user_id: int) -> MessageItemResult:
        """Помечает сообщение удалённым, сохраняя его место в истории диалога."""
        async with self.transaction_manager:
            message = await self.repository.delete_message(conversation_id, message_id, user_id)
            await self._record_message_event(MESSAGE_DELETED_EVENT, conversation_id, user_id, message_id)
        return MessageItemResult(message=message)

    async def mark_read(self, conversation_id: int, user_id: int, message_id: int) -> None:
        async with self.transaction_manager:
            updated = await self.repository.mark_read(conversation_id, user_id, message_id)
            if updated:
                await self._record_message_event(MESSAGE_READ_EVENT, conversation_id, user_id, message_id)
        if not updated:
            raise NotFoundError("Сообщение или диалог не найдены")

    async def _record_message_event(
        self,
        event_type: str,
        conversation_id: int,
        user_id: int,
        message_id: int,
    ) -> None:
        """Store message analytics in the same transaction as the mutation."""
        await self.outbox_repository.add(
            IntegrationEvent(
                event_type=event_type,
                payload=build_analytics_event_payload(
                    user_id=user_id,
                    entity_type="message",
                    entity_id=message_id,
                    data={"conversation_id": conversation_id},
                ),
            )
        )
