from dishka.integrations.fastapi import FromDishka, inject
from fastapi import APIRouter, WebSocket, WebSocketDisconnect, status

from backend.application.ports.token_service import AuthTokenService
from backend.application.ports.user_repository import UserRepository
from backend.application.use_cases.messages import MessagingUseCase
from backend.presentation.messages.ws.handlers import authenticate_websocket, handle_socket_event
from backend.presentation.messages.ws.ports import MessageConnectionManagerPort


router = APIRouter(prefix="/messages", tags=["Messages"])


@router.websocket("/ws")
@inject
async def messages_socket(
    websocket: WebSocket,
    use_case: FromDishka[MessagingUseCase],
    token_service: FromDishka[AuthTokenService],
    user_repository: FromDishka[UserRepository],
    manager: FromDishka[MessageConnectionManagerPort],
) -> None:
    """Открывает realtime-канал сообщений для авторизованного пользователя.

    WebSocket аутентифицируется по access-token в cookie. Клиент может
    подписываться на диалоги, отправлять сообщения, отмечать их прочитанными
    и поддерживать соединение ping-событиями.
    """
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
                manager,
                subscribed_conversations,
            )
    except (WebSocketDisconnect, ValueError, KeyError, TypeError):
        pass
    finally:
        manager.cleanup(websocket, subscribed_conversations)
