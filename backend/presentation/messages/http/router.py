from typing import Annotated

from dishka.integrations.fastapi import DishkaRoute, FromDishka
from fastapi import APIRouter, Query, status

from backend.application.use_cases.messages import MessagingUseCase
from backend.domain.user.entity import User
from backend.presentation.messages.auth import require_user_id
from backend.presentation.messages.http.schemas import (
    DirectConversationRequest,
    MarkReadRequest,
    SendMessageRequest,
)
from backend.presentation.messages.http.serializers import (
    ConversationPayload,
    MessagePagePayload,
    MessagePayload,
    conversation_to_payload,
    message_to_payload,
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
) -> list[ConversationPayload]:
    """Возвращает список direct-диалогов текущего авторизованного пользователя."""
    user_id = require_user_id(current_user)
    results = await use_case.list_conversations(user_id)
    return [conversation_to_payload(item.conversation, user_id) for item in results]


@router.get("/conversations/{conversation_id}/messages")
async def list_messages(
    conversation_id: int,
    use_case: FromDishka[MessagingUseCase],
    current_user: FromDishka[User],
    cursor: Annotated[str | None, Query()] = None,
    limit: Annotated[int, Query(ge=1, le=100)] = 30,
) -> MessagePagePayload:
    """Возвращает страницу сообщений диалога в обратном хронологическом порядке.

    Параметр ``cursor`` используется для загрузки следующей страницы, а
    ``limit`` ограничивает количество сообщений от 1 до 100.
    """
    result = await use_case.list_messages(conversation_id, require_user_id(current_user), cursor, limit)
    return {
        "items": [message_to_payload(item) for item in result.messages],
        "next_cursor": result.next_cursor,
        "has_more": result.has_more,
    }


@router.post("/conversations/{conversation_id}/messages")
async def send_message(
    conversation_id: int,
    body: SendMessageRequest,
    use_case: FromDishka[MessagingUseCase],
    current_user: FromDishka[User],
    manager: FromDishka[MessageConnectionManagerPort],
) -> MessagePayload:
    """Сохраняет сообщение и публикует событие для всех участников диалога."""
    result = await use_case.send_message(conversation_id, require_user_id(current_user), body.text)
    payload = message_to_payload(result.message)
    await manager.broadcast(conversation_id, {"type": "message.new", "message": payload})
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
