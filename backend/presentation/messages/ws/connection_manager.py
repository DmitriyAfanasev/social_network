from collections import defaultdict

from fastapi import WebSocket, WebSocketDisconnect

from backend.presentation.messages.ws.schemas import WsEvent


class MessageConnectionManager:
    """Manage sockets subscribed to direct-message conversations."""

    def __init__(self) -> None:
        # TODO: replace the process-local registry with Redis Pub/Sub or a broker
        # before running multiple application workers.
        self._connections: defaultdict[int, set[WebSocket]] = defaultdict(set)

    def subscribe(self, conversation_id: int, websocket: WebSocket) -> None:
        self._connections[conversation_id].add(websocket)

    def unsubscribe(self, conversation_id: int, websocket: WebSocket) -> None:
        self._connections[conversation_id].discard(websocket)
        if not self._connections[conversation_id]:
            self._connections.pop(conversation_id, None)

    async def broadcast(self, conversation_id: int, event: WsEvent) -> None:
        stale_connections: set[WebSocket] = set()
        for client in tuple(self._connections[conversation_id]):
            try:
                await client.send_json(event)
            except (RuntimeError, WebSocketDisconnect):
                stale_connections.add(client)

        for client in stale_connections:
            self.unsubscribe(conversation_id, client)

    def cleanup(self, websocket: WebSocket, conversations: set[int]) -> None:
        for conversation_id in conversations:
            self.unsubscribe(conversation_id, websocket)


message_connections = MessageConnectionManager()
