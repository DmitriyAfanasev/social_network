import asyncio
from collections.abc import Mapping
from typing import Any, Protocol

from fastapi import WebSocket

from backend.presentation.messages.ws.schemas import WsEvent


NotificationEvent = Mapping[str, Any]


class MessageConnectionManagerPort(Protocol):
    """Абстракция локальных WebSocket-подключений сообщений."""

    async def subscribe(self, conversation_id: int, websocket: WebSocket) -> None:
        """Подписывает WebSocket на диалог."""
        ...

    def unsubscribe(self, conversation_id: int, websocket: WebSocket) -> None:
        """Отписывает WebSocket от диалога."""
        ...

    async def broadcast(self, conversation_id: int, event: WsEvent) -> None:
        """Публикует событие диалога."""
        ...

    async def subscribe_notifications(self, user_id: int) -> asyncio.Queue[NotificationEvent]:
        """Подписывает пользователя на события для SSE."""
        ...

    def unsubscribe_notifications(self, user_id: int, queue: asyncio.Queue[NotificationEvent]) -> None:
        """Удаляет SSE-подписку пользователя."""
        ...

    async def publish_notification(
        self,
        recipient_ids: list[int],
        event: NotificationEvent,
    ) -> None:
        """Публикует событие в SSE-потоки получателей."""
        ...

    def cleanup(self, websocket: WebSocket, conversations: set[int]) -> None:
        """Удаляет WebSocket из всех его подписок."""
        ...
