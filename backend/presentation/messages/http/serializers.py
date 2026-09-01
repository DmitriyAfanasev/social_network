from datetime import datetime
from typing import TypedDict

from backend.application.read_models import ConversationReadModel, MessageReadModel


class ConversationPayload(TypedDict):
    id: int
    participant_ids: list[int]
    other_user_id: int | None
    created_at: datetime
    archived: bool
    pinned: bool
    muted: bool
    can_send_message: bool
    last_message: "LastMessagePayload | None"


class LastMessagePayload(TypedDict):
    text: str
    sender_id: int
    content_type: str | None
    created_at: str


class MessagePayload(TypedDict):
    id: int
    conversation_id: int
    sender_id: int
    text: str
    created_at: str
    media_id: int | None
    media_content_type: str | None
    edited_at: str | None
    deleted_at: str | None


class MessagePagePayload(TypedDict):
    items: list[MessagePayload]
    next_cursor: str | None
    has_more: bool


def conversation_to_payload(conversation: ConversationReadModel, user_id: int) -> ConversationPayload:
    """Convert a conversation model to the public HTTP response payload."""
    participant_ids = [item.user_id for item in conversation.participants]
    last_message = conversation.last_message
    return {
        "id": conversation.id,
        "participant_ids": participant_ids,
        "other_user_id": next((item for item in participant_ids if item != user_id), None),
        "created_at": conversation.created_at,
        "archived": next(
            item.archived_at is not None
            for item in conversation.participants
            if item.user_id == user_id
        ),
        "pinned": next(item.pinned_at is not None for item in conversation.participants if item.user_id == user_id),
        "muted": next(item.muted_at is not None for item in conversation.participants if item.user_id == user_id),
        "can_send_message": False,
        "last_message": (
            {
                "text": last_message.text,
                "sender_id": last_message.sender_id,
                "content_type": last_message.media.content_type if last_message.media else None,
                "created_at": last_message.created_at.isoformat(),
            }
            if last_message is not None
            else None
        ),
    }


def message_to_payload(message: MessageReadModel) -> MessagePayload:
    """Convert a message model to a JSON-safe HTTP/WebSocket payload."""
    return {
        "id": message.id,
        "conversation_id": message.conversation_id,
        "sender_id": message.sender_id,
        "text": message.text,
        "created_at": message.created_at.isoformat(),
        "media_id": message.media_id,
        "media_content_type": message.media.content_type if message.media else None,
        "edited_at": message.edited_at.isoformat() if message.edited_at else None,
        "deleted_at": message.deleted_at.isoformat() if message.deleted_at else None,
    }
