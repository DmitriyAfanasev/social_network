import base64
import json
from datetime import datetime

from backend.application.exceptions import NotFoundError, ValidationAppError
from backend.application.ports.message_repository import MessageRepository
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
    def __init__(self, repository: MessageRepository, transaction_manager: TransactionManager) -> None:
        self.repository = repository
        self.transaction_manager = transaction_manager

    async def get_or_create_direct(self, user_id: int, other_user_id: int) -> ConversationResult:
        if user_id == other_user_id:
            raise ValidationAppError("Нельзя создать диалог с самим собой")
        async with self.transaction_manager:
            conversation = await self.repository.get_or_create_direct(user_id, other_user_id)
        return ConversationResult(conversation=conversation)

    async def list_conversations(self, user_id: int) -> list[ConversationResult]:
        return [ConversationResult(conversation=item) for item in await self.repository.list_conversations(user_id)]

    async def list_messages(
        self, conversation_id: int, user_id: int, cursor: str | None, limit: int
    ) -> MessagePageResult:
        if not 1 <= limit <= 100:
            raise ValidationAppError("Параметр limit должен быть от 1 до 100")
        messages, has_more = await self.repository.list_messages(
            conversation_id, user_id, decode_cursor(cursor), limit + 1
        )
        next_cursor = encode_cursor(messages[limit - 1].created_at, messages[limit - 1].id) if has_more else None
        return MessagePageResult(messages=messages[:limit], next_cursor=next_cursor, has_more=has_more)

    async def send_message(self, conversation_id: int, user_id: int, text: str) -> MessageItemResult:
        normalized = text.strip()
        if not normalized:
            raise ValidationAppError("Сообщение не может быть пустым")
        if len(normalized) > 5000:
            raise ValidationAppError("Сообщение слишком длинное")
        async with self.transaction_manager:
            message = await self.repository.create_message(conversation_id, user_id, normalized)
        return MessageItemResult(message=message)

    async def mark_read(self, conversation_id: int, user_id: int, message_id: int) -> None:
        async with self.transaction_manager:
            updated = await self.repository.mark_read(conversation_id, user_id, message_id)
        if not updated:
            raise NotFoundError("Сообщение или диалог не найдены")
