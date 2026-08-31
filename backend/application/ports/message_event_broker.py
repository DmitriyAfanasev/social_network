from collections.abc import Awaitable, Callable, Mapping
from typing import Any, Protocol


MessageEvent = Mapping[str, Any]
MessageEventHandler = Callable[[int, MessageEvent], Awaitable[None]]


class MessageEventBroker(Protocol):
    """Абстракция публикации и получения событий сообщений."""

    async def start(self, handler: MessageEventHandler) -> None:
        """Запускает подписку на события и передаёт их обработчику."""
        ...

    async def publish(self, conversation_id: int, event: MessageEvent) -> None:
        """Публикует событие для участников диалога."""
        ...

    async def close(self) -> None:
        """Освобождает ресурсы брокера событий."""
        ...
