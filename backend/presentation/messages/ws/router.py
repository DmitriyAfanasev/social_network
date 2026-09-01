from dishka.integrations.fastapi import FromDishka, inject
from fastapi import APIRouter, WebSocket, WebSocketDisconnect, status

from backend.application.exceptions import PermissionDeniedError
from backend.application.use_cases.auth import GetCurrentUserUseCase
from backend.application.use_cases.messages import MessagingUseCase
from backend.application.use_cases.users import TouchUserActivityUseCase
from backend.presentation.messages.ws.constants import WEBSOCKET_ERROR_EVENT
from backend.presentation.messages.ws.handlers import authenticate_websocket, handle_socket_event
from backend.presentation.messages.ws.ports import MessageConnectionManagerPort


router = APIRouter(prefix="/messages", tags=["Messages"])


@router.websocket("/ws")
@inject
async def messages_socket(
    websocket: WebSocket,
    use_case: FromDishka[MessagingUseCase],
    current_user_use_case: FromDishka[GetCurrentUserUseCase],
    touch_user_activity: FromDishka[TouchUserActivityUseCase],
    manager: FromDishka[MessageConnectionManagerPort],
) -> None:
    """Открывает realtime-канал сообщений для авторизованного пользователя.

    WebSocket аутентифицируется по access-token в cookie. Клиент может
    подписываться на диалоги, отправлять сообщения, отмечать их прочитанными
    и поддерживать соединение ping-событиями.
    """
    user_id = await authenticate_websocket(websocket, current_user_use_case)
    if user_id is None:
        await websocket.close(code=status.WS_1008_POLICY_VIOLATION)
        return

    await touch_user_activity.execute(user_id)
    await websocket.accept()
    subscribed_conversations: set[int] = set()
    try:
        while True:
            event = await websocket.receive_json()
            try:
                await handle_socket_event(
                    websocket,
                    event,
                    user_id,
                    use_case,
                    manager,
                    touch_user_activity,
                    subscribed_conversations,
                )
            except PermissionDeniedError as error:
                await websocket.send_json(
                    {"type": WEBSOCKET_ERROR_EVENT, "message": error.message}
                )
    except (WebSocketDisconnect, ValueError, KeyError, TypeError):
        pass
    finally:
        manager.cleanup(websocket, subscribed_conversations)
