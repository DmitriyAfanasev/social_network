from typing import TypedDict

from backend.presentation.messages.http.serializers import MessagePayload


class WsMessageNew(TypedDict):
    type: str
    message: MessagePayload


class WsConversationSubscribed(TypedDict):
    type: str
    conversation_id: int


class WsMessageRead(TypedDict):
    type: str
    conversation_id: int
    message_id: int
    user_id: int


class WsError(TypedDict):
    type: str
    message: str


WsEvent = WsMessageNew | WsConversationSubscribed | WsMessageRead | WsError
