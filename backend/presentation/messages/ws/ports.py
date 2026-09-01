import asyncio
from collections.abc import Mapping
from typing import Any, Protocol

from backend.presentation.messages.ws.schemas import WsEvent


NotificationEvent = Mapping[str, Any]
type WebSocketPayload = dict[str, Any]


class WebSocketSender(Protocol):
    """Minimal WebSocket contract required by message handlers."""

    @property
    def cookies(self) -> Mapping[str, str]:
        ...

    async def send_json(self, data: Any) -> None:
        ...


class MessageConnectionManagerPort(Protocol):
    """Абстракция локальных WebSocket-подключений сообщений."""

    async def subscribe(self, conversation_id: int, websocket: WebSocketSender) -> None:
        """Подписывает WebSocket на диалог."""
        ...

    def unsubscribe(self, conversation_id: int, websocket: WebSocketSender) -> None:
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

    def cleanup(self, websocket: WebSocketSender, conversations: set[int]) -> None:
        """Удаляет WebSocket из всех его подписок."""
        ...
