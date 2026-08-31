from typing import Literal, TypedDict

from backend.presentation.messages.http.serializers import MessagePayload


class WsMessageNew(TypedDict):
    type: Literal["message.new"]
    message: MessagePayload


class WsConversationSubscribed(TypedDict):
    type: Literal["conversation.subscribed"]
    conversation_id: int


class WsMessageRead(TypedDict):
    type: Literal["message.read"]
    conversation_id: int
    message_id: int
    user_id: int


class WsError(TypedDict):
    type: Literal["error"]
    message: str


WsEvent = WsMessageNew | WsConversationSubscribed | WsMessageRead | WsError
