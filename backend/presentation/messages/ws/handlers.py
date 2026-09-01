from collections.abc import Mapping
from typing import Any

from backend.application.use_cases.auth import GetCurrentUserUseCase
from backend.application.use_cases.messages import MessagingUseCase
from backend.application.use_cases.users import TouchUserActivityUseCase
from backend.presentation.messages.auth import require_user_id
from backend.presentation.messages.http.serializers import message_to_payload
from backend.presentation.messages.ws.constants import (
    CONVERSATION_SUBSCRIBE_EVENT,
    CONVERSATION_SUBSCRIBED_EVENT,
    MESSAGE_NEW_EVENT,
    MESSAGE_READ_EVENT,
    MESSAGE_SEND_EVENT,
    TYPING_EVENT,
    WEBSOCKET_ERROR_EVENT,
    WEBSOCKET_PING_EVENT,
    WEBSOCKET_PONG_EVENT,
)
from backend.presentation.messages.ws.ports import MessageConnectionManagerPort, WebSocketSender


async def authenticate_websocket(
    websocket: WebSocketSender,
    current_user_use_case: GetCurrentUserUseCase,
) -> int | None:
    """Authenticate a WebSocket using the access token from its cookies."""
    token = websocket.cookies.get("access-token")
    if not token:
        return None

    user = await current_user_use_case.execute(token, required=False)
    if user is None:
        return None
    return require_user_id(user)


async def subscribe_to_conversation(
    websocket: WebSocketSender,
    conversation_id: int,
    user_id: int,
    use_case: MessagingUseCase,
    manager: MessageConnectionManagerPort,
    subscribed_conversations: set[int],
) -> None:
    """Verify conversation membership and subscribe the socket to it."""
    await use_case.ensure_conversation_access(conversation_id, user_id)
    await use_case.list_messages(conversation_id, user_id, 0, 1)
    await manager.subscribe(conversation_id, websocket)
    subscribed_conversations.add(conversation_id)
    await websocket.send_json({"type": CONVERSATION_SUBSCRIBED_EVENT, "conversation_id": conversation_id})


async def handle_socket_event(
    websocket: WebSocketSender,
    event: Mapping[str, Any],
    user_id: int,
    use_case: MessagingUseCase,
    manager: MessageConnectionManagerPort,
    touch_user_activity: TouchUserActivityUseCase,
    subscribed_conversations: set[int],
) -> None:
    """Dispatch one validated-enough WebSocket event to its handler."""
    event_type = event.get("type")
    conversation_id = int(event.get("conversation_id", 0))

    if event_type == CONVERSATION_SUBSCRIBE_EVENT:
        await subscribe_to_conversation(
            websocket, conversation_id, user_id, use_case, manager, subscribed_conversations
        )
    elif event_type == MESSAGE_SEND_EVENT:
        result = await use_case.send_message(conversation_id, user_id, str(event.get("text", "")))
        recipient_ids = await use_case.get_participant_ids(conversation_id)
        await manager.broadcast(
            conversation_id,
            {
                "type": MESSAGE_NEW_EVENT,
                "message": message_to_payload(result.message),
                "recipient_ids": recipient_ids,
            },
        )
    elif event_type == "message.read":
        message_id = int(event["message_id"])
        await use_case.mark_read(conversation_id, user_id, message_id)
        recipient_ids = await use_case.get_participant_ids(conversation_id)
        await manager.broadcast(
            conversation_id,
            {
                "type": MESSAGE_READ_EVENT,
                "conversation_id": conversation_id,
                "message_id": message_id,
                "user_id": user_id,
                "recipient_ids": recipient_ids,
            },
        )
    elif event_type == TYPING_EVENT:
        if conversation_id not in subscribed_conversations:
            return
        await use_case.ensure_conversation_access(conversation_id, user_id)
        recipient_ids = await use_case.get_participant_ids(conversation_id)
        await manager.broadcast(
            conversation_id,
            {
                "type": TYPING_EVENT,
                "conversation_id": conversation_id,
                "user_id": user_id,
                "is_typing": bool(event.get("is_typing", False)),
                "recipient_ids": recipient_ids,
            },
        )
    elif event_type == WEBSOCKET_PING_EVENT:
        await touch_user_activity.execute(user_id)
        await websocket.send_json({"type": WEBSOCKET_PONG_EVENT})
    else:
        await websocket.send_json({"type": WEBSOCKET_ERROR_EVENT, "message": "Неизвестный тип события"})
