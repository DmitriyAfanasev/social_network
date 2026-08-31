from typing import Protocol

from fastapi import WebSocket

from backend.presentation.messages.ws.schemas import WsEvent


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

    def cleanup(self, websocket: WebSocket, conversations: set[int]) -> None:
        """Удаляет WebSocket из всех его подписок."""
        ...
