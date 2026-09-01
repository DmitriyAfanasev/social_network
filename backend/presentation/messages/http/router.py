from typing import Annotated

from dishka.integrations.fastapi import DishkaRoute, FromDishka
from fastapi import APIRouter, File, Form, Query, UploadFile, status

from backend.application.ports.media_storage import MediaStorage
from backend.application.use_cases.messages import MessagingUseCase
from backend.domain.user.entity import User
from backend.presentation.messages.auth import require_user_id
from backend.presentation.messages.http.schemas import (
    DirectConversationRequest,
    EditMessageRequest,
    MarkReadRequest,
)
from backend.presentation.messages.http.serializers import (
    ConversationPayload,
    MessagePagePayload,
    MessagePayload,
    conversation_to_payload,
    message_to_payload,
)
from backend.presentation.messages.ws.constants import (
    MESSAGE_DELETED_EVENT,
    MESSAGE_NEW_EVENT,
    MESSAGE_UPDATED_EVENT,
)
from backend.presentation.messages.ws.ports import MessageConnectionManagerPort


router = APIRouter(prefix="/messages", tags=["Messages"], route_class=DishkaRoute)


@router.post("/conversations/direct", status_code=status.HTTP_200_OK)
async def create_direct_conversation(
    body: DirectConversationRequest,
    use_case: FromDishka[MessagingUseCase],
    current_user: FromDishka[User],
) -> ConversationPayload:
    """Создаёт или возвращает существующий direct-диалог с пользователем.

    Если диалог между текущим пользователем и указанным пользователем уже
    существует, возвращается его текущая запись. Иначе создаётся новый диалог.
    """
    user_id = require_user_id(current_user)
    result = await use_case.get_or_create_direct(user_id, body.user_id)
    return conversation_to_payload(result.conversation, user_id)


@router.get("/conversations")
async def list_conversations(
    use_case: FromDishka[MessagingUseCase],
    current_user: FromDishka[User],
    archived: bool = False,
) -> list[ConversationPayload]:
    """Возвращает список direct-диалогов текущего авторизованного пользователя."""
    user_id = require_user_id(current_user)
    results = await use_case.list_conversations(user_id, archived=archived)
    conversations = []
    for item in results:
        payload = conversation_to_payload(item.conversation, user_id)
        other_user_id = payload["other_user_id"]
        payload["can_send_message"] = (
            True
            if other_user_id is None
            else await use_case.can_send_message(user_id, other_user_id)
        )
        conversations.append(payload)
    return conversations


@router.patch("/conversations/{conversation_id}/archive", status_code=status.HTTP_204_NO_CONTENT)
async def archive_conversation(
    conversation_id: int,
    use_case: FromDishka[MessagingUseCase],
    current_user: FromDishka[User],
) -> None:
    await use_case.archive_conversation(conversation_id, require_user_id(current_user))


@router.patch("/conversations/{conversation_id}/unarchive", status_code=status.HTTP_204_NO_CONTENT)
async def unarchive_conversation(
    conversation_id: int,
    use_case: FromDishka[MessagingUseCase],
    current_user: FromDishka[User],
) -> None:
    await use_case.unarchive_conversation(conversation_id, require_user_id(current_user))


@router.delete("/conversations/{conversation_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_conversation(
    conversation_id: int,
    use_case: FromDishka[MessagingUseCase],
    current_user: FromDishka[User],
) -> None:
    await use_case.hide_conversation(conversation_id, require_user_id(current_user))


@router.post("/conversations/{conversation_id}/unread", status_code=status.HTTP_204_NO_CONTENT)
async def mark_conversation_unread(
    conversation_id: int,
    use_case: FromDishka[MessagingUseCase],
    current_user: FromDishka[User],
) -> None:
    await use_case.mark_unread(conversation_id, require_user_id(current_user))


@router.patch("/conversations/{conversation_id}/pin", status_code=status.HTTP_204_NO_CONTENT)
async def pin_conversation(
    conversation_id: int,
    use_case: FromDishka[MessagingUseCase],
    current_user: FromDishka[User],
) -> None:
    await use_case.pin_conversation(conversation_id, require_user_id(current_user), True)


@router.patch("/conversations/{conversation_id}/unpin", status_code=status.HTTP_204_NO_CONTENT)
async def unpin_conversation(
    conversation_id: int,
    use_case: FromDishka[MessagingUseCase],
    current_user: FromDishka[User],
) -> None:
    await use_case.pin_conversation(conversation_id, require_user_id(current_user), False)


@router.patch("/conversations/{conversation_id}/mute", status_code=status.HTTP_204_NO_CONTENT)
async def mute_conversation(
    conversation_id: int,
    use_case: FromDishka[MessagingUseCase],
    current_user: FromDishka[User],
) -> None:
    await use_case.mute_conversation(conversation_id, require_user_id(current_user), True)


@router.patch("/conversations/{conversation_id}/unmute", status_code=status.HTTP_204_NO_CONTENT)
async def unmute_conversation(
    conversation_id: int,
    use_case: FromDishka[MessagingUseCase],
    current_user: FromDishka[User],
) -> None:
    await use_case.mute_conversation(conversation_id, require_user_id(current_user), False)


@router.delete("/conversations/{conversation_id}/history", status_code=status.HTTP_204_NO_CONTENT)
async def clear_conversation_history(
    conversation_id: int,
    use_case: FromDishka[MessagingUseCase],
    current_user: FromDishka[User],
) -> None:
    await use_case.clear_history(conversation_id, require_user_id(current_user))


@router.get("/conversations/{conversation_id}/messages")
async def list_messages(
    conversation_id: int,
    use_case: FromDishka[MessagingUseCase],
    current_user: FromDishka[User],
    offset: Annotated[int, Query(ge=0)] = 0,
    limit: Annotated[int, Query(ge=1, le=100)] = 30,
) -> MessagePagePayload:
    """Возвращает страницу сообщений диалога в обратном хронологическом порядке.

    Параметр ``offset`` задаёт смещение страницы, а ``limit`` ограничивает
    количество сообщений от 1 до 100.
    """
    result = await use_case.list_messages(conversation_id, require_user_id(current_user), offset, limit)
    return {
        "items": [message_to_payload(item) for item in result.messages],
        "next_cursor": result.next_cursor,
        "has_more": result.has_more,
    }


@router.post("/conversations/{conversation_id}/messages")
async def send_message(
    conversation_id: int,
    use_case: FromDishka[MessagingUseCase],
    current_user: FromDishka[User],
    manager: FromDishka[MessageConnectionManagerPort],
    media_storage: FromDishka[MediaStorage],
    text: str = Form("", max_length=5000),
    media: UploadFile | None = File(None),
) -> MessagePayload:
    """Сохраняет текст и/или медиа сообщения и уведомляет участников диалога."""
    media_item = None
    if media is not None and media.filename:
        media_item = await media_storage.upload(
            file=media,
            directory=f"messages/{conversation_id}",
            uploaded_by=require_user_id(current_user),
        )
    result = await use_case.send_message(
        conversation_id,
        require_user_id(current_user),
        text,
        media_item.id if media_item else None,
    )
    payload = message_to_payload(result.message)
    recipient_ids = await use_case.get_participant_ids(conversation_id)
    await manager.broadcast(
        conversation_id,
        {"type": MESSAGE_NEW_EVENT, "message": payload, "recipient_ids": recipient_ids},
    )
    return payload


@router.patch("/conversations/{conversation_id}/messages/{message_id}")
async def edit_message(
    conversation_id: int,
    message_id: int,
    body: EditMessageRequest,
    use_case: FromDishka[MessagingUseCase],
    current_user: FromDishka[User],
    manager: FromDishka[MessageConnectionManagerPort],
) -> MessagePayload:
    """Изменяет текст сообщения автора и рассылает событие `message.updated`."""
    result = await use_case.edit_message(conversation_id, message_id, require_user_id(current_user), body.text)
    payload = message_to_payload(result.message)
    recipient_ids = await use_case.get_participant_ids(conversation_id)
    await manager.broadcast(
        conversation_id,
        {"type": MESSAGE_UPDATED_EVENT, "message": payload, "recipient_ids": recipient_ids},
    )
    return payload


@router.delete("/conversations/{conversation_id}/messages/{message_id}/media")
async def remove_message_media(
    conversation_id: int,
    message_id: int,
    use_case: FromDishka[MessagingUseCase],
    current_user: FromDishka[User],
    manager: FromDishka[MessageConnectionManagerPort],
) -> MessagePayload:
    result = await use_case.remove_message_media(conversation_id, message_id, require_user_id(current_user))
    payload = message_to_payload(result.message)
    recipient_ids = await use_case.get_participant_ids(conversation_id)
    await manager.broadcast(
        conversation_id,
        {"type": MESSAGE_UPDATED_EVENT, "message": payload, "recipient_ids": recipient_ids},
    )
    return payload



@router.delete("/conversations/{conversation_id}/messages/{message_id}")
async def delete_message(
    conversation_id: int,
    message_id: int,
    use_case: FromDishka[MessagingUseCase],
    current_user: FromDishka[User],
    manager: FromDishka[MessageConnectionManagerPort],
) -> MessagePayload:
    """Удаляет сообщение автора и рассылает событие `message.deleted`."""
    result = await use_case.delete_message(conversation_id, message_id, require_user_id(current_user))
    payload = message_to_payload(result.message)
    recipient_ids = await use_case.get_participant_ids(conversation_id)
    await manager.broadcast(
        conversation_id,
        {"type": MESSAGE_DELETED_EVENT, "message": payload, "recipient_ids": recipient_ids},
    )
    return payload


@router.post("/conversations/{conversation_id}/read", status_code=status.HTTP_204_NO_CONTENT)
async def mark_read(
    conversation_id: int,
    body: MarkReadRequest,
    use_case: FromDishka[MessagingUseCase],
    current_user: FromDishka[User],
) -> None:
    """Отмечает указанное сообщение диалога прочитанным текущим пользователем."""
    await use_case.mark_read(conversation_id, require_user_id(current_user), body.message_id)
