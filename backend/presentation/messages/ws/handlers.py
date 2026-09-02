from collections.abc import Mapping
from typing import Any

from backend.application.dto import CallDTO
from backend.application.events import JsonValue
from backend.application.use_cases.auth import GetCurrentUserUseCase
from backend.application.use_cases.calls import CallUseCase
from backend.application.use_cases.messages import MessagingUseCase
from backend.application.use_cases.users import TouchUserActivityUseCase
from backend.domain.call import CallType
from backend.presentation.messages.auth import require_user_id
from backend.presentation.messages.http.serializers import message_to_payload
from backend.presentation.messages.ws.constants import (
    CALL_ACCEPT_EVENT,
    CALL_END_EVENT,
    CALL_INVITE_EVENT,
    CALL_REJECT_EVENT,
    CALL_SIGNAL_EVENT,
    CALL_START_EVENT,
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
from backend.presentation.messages.ws.schemas import WsCallEvent


def _call_event(
    event_type: str,
    call: CallDTO,
    sender_id: int,
    recipient_ids: list[int],
    signal: dict[str, JsonValue] | None = None,
) -> WsCallEvent:
    return {
        "type": event_type,
        "sender_id": sender_id,
        "call_id": call.call_id,
        "caller_id": call.caller_id,
        "callee_id": call.callee_id,
        "call_type": call.call_type,
        "status": call.status,
        "recipient_ids": recipient_ids,
        "signal": signal,
    }


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
    call_use_case: CallUseCase | None = None,
) -> None:
    """Dispatch one validated-enough WebSocket event to its handler."""
    event_type = event.get("type")
    conversation_id = int(event.get("conversation_id", 0))

    if event_type in {
        CALL_START_EVENT,
        CALL_ACCEPT_EVENT,
        CALL_REJECT_EVENT,
        CALL_END_EVENT,
        CALL_SIGNAL_EVENT,
    }:
        if call_use_case is None:
            raise RuntimeError("Call use case is not configured")
        if conversation_id not in subscribed_conversations:
            return
        recipient_ids = await use_case.get_participant_ids(conversation_id)
        if event_type == CALL_START_EVENT:
            call = await call_use_case.start(
                user_id,
                int(event["target_user_id"]),
                CallType(str(event.get("call_type", CallType.AUDIO))),
            )
            outgoing_type = CALL_INVITE_EVENT
        elif event_type == CALL_ACCEPT_EVENT:
            call = await call_use_case.accept(str(event["call_id"]), user_id)
            outgoing_type = CALL_ACCEPT_EVENT
        elif event_type == CALL_REJECT_EVENT:
            call = await call_use_case.reject(str(event["call_id"]), user_id)
            outgoing_type = CALL_REJECT_EVENT
        elif event_type == CALL_END_EVENT:
            call = await call_use_case.end(str(event["call_id"]), user_id)
            outgoing_type = CALL_END_EVENT
        else:
            call = await call_use_case.authorize_signaling(str(event["call_id"]), user_id)
            outgoing_type = CALL_SIGNAL_EVENT
        signal = event.get("signal")
        await manager.broadcast(
            conversation_id,
            _call_event(
                outgoing_type,
                call,
                user_id,
                recipient_ids,
                signal if isinstance(signal, dict) else None,
            ),
        )
    elif event_type == CONVERSATION_SUBSCRIBE_EVENT:
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
