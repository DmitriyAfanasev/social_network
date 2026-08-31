from dishka.integrations.fastapi import FromDishka, inject
from fastapi import APIRouter, WebSocket, WebSocketDisconnect, status

from backend.application.ports.token_service import AuthTokenService
from backend.application.ports.user_repository import UserRepository
from backend.application.use_cases.messages import MessagingUseCase
from backend.presentation.messages.ws.connection_manager import message_connections
from backend.presentation.messages.ws.handlers import authenticate_websocket, handle_socket_event


router = APIRouter(prefix="/messages", tags=["Messages"])


@router.websocket("/ws")
@inject
async def messages_socket(
    websocket: WebSocket,
    use_case: FromDishka[MessagingUseCase],
    token_service: FromDishka[AuthTokenService],
    user_repository: FromDishka[UserRepository],
) -> None:
    """Authenticate a socket and dispatch its incoming messaging events."""
    user_id = await authenticate_websocket(websocket, token_service, user_repository)
    if user_id is None:
        await websocket.close(code=status.WS_1008_POLICY_VIOLATION)
        return

    await websocket.accept()
    subscribed_conversations: set[int] = set()
    try:
        while True:
            event = await websocket.receive_json()
            await handle_socket_event(
                websocket,
                event,
                user_id,
                use_case,
                message_connections,
                subscribed_conversations,
            )
    except (WebSocketDisconnect, ValueError, KeyError, TypeError):
        pass
    finally:
        message_connections.cleanup(websocket, subscribed_conversations)
