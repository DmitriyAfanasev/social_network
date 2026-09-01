from datetime import datetime
from typing import TypedDict, cast

from backend.infra.models.sqlalchemy import Conversation, Message


class ConversationPayload(TypedDict):
    id: int
    participant_ids: list[int]
    other_user_id: int | None
    created_at: datetime


class MessagePayload(TypedDict):
    id: int
    conversation_id: int
    sender_id: int
    text: str
    created_at: str
    media_id: int | None
    edited_at: str | None
    deleted_at: str | None


class MessagePagePayload(TypedDict):
    items: list[MessagePayload]
    next_cursor: str | None
    has_more: bool


def conversation_to_payload(conversation: object, user_id: int) -> ConversationPayload:
    """Convert a conversation model to the public HTTP response payload."""
    model = cast(Conversation, conversation)
    participant_ids = [item.user_id for item in model.participants]
    return {
        "id": model.id,
        "participant_ids": participant_ids,
        "other_user_id": next((item for item in participant_ids if item != user_id), None),
        "created_at": model.created_at,
    }


def message_to_payload(message: object) -> MessagePayload:
    """Convert a message model to a JSON-safe HTTP/WebSocket payload."""
    model = cast(Message, message)
    return {
        "id": model.id,
        "conversation_id": model.conversation_id,
        "sender_id": model.sender_id,
        "text": model.text,
        "created_at": model.created_at.isoformat(),
        "media_id": model.media_id,
        "edited_at": model.edited_at.isoformat() if model.edited_at else None,
        "deleted_at": model.deleted_at.isoformat() if model.deleted_at else None,
    }
