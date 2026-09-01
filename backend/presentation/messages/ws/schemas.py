from typing import TypedDict

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


WsEvent = WsMessageNew | WsMessageUpdated | WsMessageDeleted | WsConversationSubscribed | WsMessageRead | WsError
