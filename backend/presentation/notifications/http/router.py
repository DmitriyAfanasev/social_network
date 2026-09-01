import asyncio
import json
from collections.abc import AsyncIterator

from dishka.integrations.fastapi import DishkaRoute, FromDishka
from fastapi import APIRouter
from fastapi.responses import StreamingResponse

from backend.domain.user.entity import User
from backend.presentation.messages.auth import require_user_id
from backend.presentation.messages.ws.ports import (
    MessageConnectionManagerPort,
    NotificationEvent,
)


router = APIRouter(prefix="/notifications", tags=["Notifications"], route_class=DishkaRoute)


@router.get("/stream")
async def notification_stream(
    current_user: FromDishka[User],
    manager: FromDishka[MessageConnectionManagerPort],
) -> StreamingResponse:
    """Открывает SSE-поток уведомлений текущего пользователя."""
    user_id = require_user_id(current_user)
    queue = await manager.subscribe_notifications(user_id)

    async def events() -> AsyncIterator[str]:
        try:
            yield "retry: 3000\n\n"
            while True:
                try:
                    event: NotificationEvent = await asyncio.wait_for(queue.get(), timeout=15)
                except TimeoutError:
                    yield ": keep-alive\n\n"
                    continue
                yield f"event: {event.get('type', 'notification')}\ndata: {json.dumps(event, ensure_ascii=False)}\n\n"
        finally:
            manager.unsubscribe_notifications(user_id, queue)

    return StreamingResponse(
        events(),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "Connection": "keep-alive",
            "X-Accel-Buffering": "no",
        },
    )
