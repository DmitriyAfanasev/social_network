from collections import defaultdict
from typing import cast

from fastapi import WebSocket, WebSocketDisconnect

from backend.application.ports.message_event_broker import MessageEvent, MessageEventBroker
from backend.presentation.messages.ws.ports import MessageConnectionManagerPort
from backend.presentation.messages.ws.schemas import WsEvent


class MessageConnectionManager(MessageConnectionManagerPort):
    """Управляет локальными WebSocket-подключениями сообщений."""

    def __init__(self, event_broker: MessageEventBroker) -> None:
        """Создаёт менеджер поверх абстрактного брокера событий."""
        self._connections: defaultdict[int, set[WebSocket]] = defaultdict(set)
        self._event_broker = event_broker

    async def _send_local(self, conversation_id: int, event: WsEvent) -> None:
        """Доставляет событие локальным клиентам выбранного диалога."""
        stale_connections: set[WebSocket] = set()
        for client in tuple(self._connections[conversation_id]):
            try:
                await client.send_json(event)
            except (RuntimeError, WebSocketDisconnect):
                stale_connections.add(client)

        for client in stale_connections:
            self.unsubscribe(conversation_id, client)

    async def subscribe(self, conversation_id: int, websocket: WebSocket) -> None:
        """Подписывает WebSocket на диалог и запускает доставку событий."""
        await self._event_broker.start(self._handle_broker_event)
        self._connections[conversation_id].add(websocket)

    def unsubscribe(self, conversation_id: int, websocket: WebSocket) -> None:
        """Удаляет WebSocket из подписки на диалог."""
        self._connections[conversation_id].discard(websocket)
        if not self._connections[conversation_id]:
            self._connections.pop(conversation_id, None)

    async def broadcast(self, conversation_id: int, event: WsEvent) -> None:
        """Передаёт событие брокеру для доставки во все процессы."""
        await self._event_broker.publish(conversation_id, event)

    async def _handle_broker_event(self, conversation_id: int, event: MessageEvent) -> None:
        """Приводит внешнее событие к WebSocket payload и доставляет его локально."""
        await self._send_local(conversation_id, cast(WsEvent, event))

    async def close(self) -> None:
        """Останавливает используемый брокер событий."""
        await self._event_broker.close()

    def cleanup(self, websocket: WebSocket, conversations: set[int]) -> None:
        """Удаляет отключившийся WebSocket из всех его диалогов."""
        for conversation_id in conversations:
            self.unsubscribe(conversation_id, websocket)
