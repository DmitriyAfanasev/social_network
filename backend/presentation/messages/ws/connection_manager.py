import asyncio
from collections import defaultdict
from contextlib import suppress
from typing import cast

from fastapi import WebSocketDisconnect

from backend.application.ports.message_event_broker import MessageEvent, MessageEventBroker
from backend.presentation.messages.ws.ports import (
    MessageConnectionManagerPort,
    NotificationEvent,
    WebSocketSender,
)
from backend.presentation.messages.ws.schemas import WsEvent


class MessageConnectionManager(MessageConnectionManagerPort):
    """Управляет локальными WebSocket-подключениями сообщений."""

    def __init__(self, event_broker: MessageEventBroker) -> None:
        """Создаёт менеджер поверх абстрактного брокера событий."""
        self._connections: defaultdict[int, set[WebSocketSender]] = defaultdict(set)
        self._notification_queues: defaultdict[int, set[asyncio.Queue[NotificationEvent]]] = defaultdict(set)
        self._event_broker = event_broker

    async def _send_local(self, conversation_id: int, event: WsEvent) -> None:
        """Доставляет событие локальным клиентам выбранного диалога."""
        stale_connections: set[WebSocketSender] = set()
        for client in tuple(self._connections[conversation_id]):
            try:
                await client.send_json(event)
            except (RuntimeError, WebSocketDisconnect):
                stale_connections.add(client)

        for client in stale_connections:
            self.unsubscribe(conversation_id, client)

    async def subscribe(self, conversation_id: int, websocket: WebSocketSender) -> None:
        """Подписывает WebSocket на диалог и запускает доставку событий."""
        await self._event_broker.start(self._handle_broker_event)
        self._connections[conversation_id].add(websocket)

    async def subscribe_notifications(self, user_id: int) -> asyncio.Queue[NotificationEvent]:
        await self._event_broker.start(self._handle_broker_event)
        queue: asyncio.Queue[NotificationEvent] = asyncio.Queue(maxsize=50)
        self._notification_queues[user_id].add(queue)
        return queue

    def unsubscribe_notifications(self, user_id: int, queue: asyncio.Queue[NotificationEvent]) -> None:
        self._notification_queues[user_id].discard(queue)
        if not self._notification_queues[user_id]:
            self._notification_queues.pop(user_id, None)

    def unsubscribe(self, conversation_id: int, websocket: WebSocketSender) -> None:
        """Удаляет WebSocket из подписки на диалог."""
        self._connections[conversation_id].discard(websocket)
        if not self._connections[conversation_id]:
            self._connections.pop(conversation_id, None)

    async def broadcast(self, conversation_id: int, event: WsEvent) -> None:
        """Передаёт событие брокеру для доставки во все процессы."""
        await self._event_broker.publish(conversation_id, event)

    async def publish_notification(
        self,
        recipient_ids: list[int],
        event: NotificationEvent,
    ) -> None:
        """Публикует событие в SSE-потоки пользователей во всех процессах."""
        await self._event_broker.publish(
            0,
            {**event, "recipient_ids": recipient_ids},
        )

    async def _handle_broker_event(self, conversation_id: int, event: MessageEvent) -> None:
        """Приводит внешнее событие к WebSocket payload и доставляет его локально."""
        await self._send_local(conversation_id, cast(WsEvent, event))
        recipient_ids = event.get("recipient_ids", [])
        for user_id in recipient_ids if isinstance(recipient_ids, list) else []:
            for queue in tuple(self._notification_queues.get(int(user_id), ())):
                with suppress(asyncio.QueueFull):
                    queue.put_nowait(event)
                    # A stale client will receive the next reconnect/update;
                    # never let one slow SSE consumer block message delivery.

    async def close(self) -> None:
        """Останавливает используемый брокер событий."""
        await self._event_broker.close()

    def cleanup(self, websocket: WebSocketSender, conversations: set[int]) -> None:
        """Удаляет отключившийся WebSocket из всех его диалогов."""
        for conversation_id in conversations:
            self.unsubscribe(conversation_id, websocket)
