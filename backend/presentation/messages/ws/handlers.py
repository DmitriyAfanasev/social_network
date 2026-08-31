from collections.abc import Mapping
from typing import Any

from fastapi import WebSocket

from backend.application.ports.token_service import AuthTokenService
from backend.application.ports.user_repository import UserRepository
from backend.application.use_cases.messages import MessagingUseCase
from backend.presentation.messages.auth import require_user_id
from backend.presentation.messages.http.serializers import message_to_payload
from backend.presentation.messages.ws.connection_manager import MessageConnectionManager


async def authenticate_websocket(
    websocket: WebSocket,
    token_service: AuthTokenService,
    user_repository: UserRepository,
) -> int | None:
    token = websocket.cookies.get("access-token")
    if not token:
        return None

    try:
        user_id = token_service.get_access_user_id(token)
        user = await user_repository.get_by_id(user_id)
    except ValueError:
        return None

    if user is None or not user.is_active:
        return None
    return require_user_id(user)


async def subscribe_to_conversation(
    websocket: WebSocket,
    conversation_id: int,
    user_id: int,
    use_case: MessagingUseCase,
    manager: MessageConnectionManager,
    subscribed_conversations: set[int],
) -> None:
    await use_case.list_messages(conversation_id, user_id, None, 1)
    manager.subscribe(conversation_id, websocket)
    subscribed_conversations.add(conversation_id)
    await websocket.send_json({"type": "conversation.subscribed", "conversation_id": conversation_id})


async def handle_socket_event(
    websocket: WebSocket,
    event: Mapping[str, Any],
    user_id: int,
    use_case: MessagingUseCase,
    manager: MessageConnectionManager,
    subscribed_conversations: set[int],
) -> None:
    event_type = event.get("type")
    conversation_id = int(event.get("conversation_id", 0))

    if event_type == "conversation.subscribe":
        await subscribe_to_conversation(
            websocket, conversation_id, user_id, use_case, manager, subscribed_conversations
        )
    elif event_type == "message.send":
        result = await use_case.send_message(conversation_id, user_id, str(event.get("text", "")))
        await manager.broadcast(conversation_id, {"type": "message.new", "message": message_to_payload(result.message)})
    elif event_type == "message.read":
        message_id = int(event["message_id"])
        await use_case.mark_read(conversation_id, user_id, message_id)
        await manager.broadcast(
            conversation_id,
            {
                "type": "message.read",
                "conversation_id": conversation_id,
                "message_id": message_id,
                "user_id": user_id,
            },
        )
    elif event_type == "ping":
        await websocket.send_json({"type": "pong"})
    else:
        await websocket.send_json({"type": "error", "message": "Неизвестный тип события"})
