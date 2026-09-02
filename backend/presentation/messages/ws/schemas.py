from typing import TypedDict

from backend.application.events import JsonValue
from backend.presentation.messages.http.serializers import MessagePayload


class WsMessageNew(TypedDict):
    type: str
    message: MessagePayload
    recipient_ids: list[int]


class WsMessageUpdated(TypedDict):
    """Событие изменения сообщения."""

    type: str
    message: MessagePayload
    recipient_ids: list[int]


class WsMessageDeleted(TypedDict):
    """Событие удаления сообщения."""

    type: str
    message: MessagePayload
    recipient_ids: list[int]


class WsConversationSubscribed(TypedDict):
    type: str
    conversation_id: int


class WsMessageRead(TypedDict):
    type: str
    conversation_id: int
    message_id: int
    user_id: int
    recipient_ids: list[int]


class WsError(TypedDict):
    type: str
    message: str


class WsTyping(TypedDict):
    type: str
    conversation_id: int
    user_id: int
    is_typing: bool
    recipient_ids: list[int]


class WsCallEvent(TypedDict):
    type: str
    sender_id: int
    call_id: str
    caller_id: int
    callee_id: int
    call_type: str
    status: str
    recipient_ids: list[int]
    signal: dict[str, JsonValue] | None


WsEvent = WsMessageNew | WsMessageUpdated | WsMessageDeleted | WsConversationSubscribed | WsMessageRead | WsTyping | WsCallEvent | WsError
