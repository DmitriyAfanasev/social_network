import asyncio
import json
from typing import Any, cast

import redis.asyncio as redis

from backend.application.ports.message_event_broker import MessageEvent, MessageEventHandler
from backend.infra.config import RedisConfig


class RedisMessageEventBroker:
    """Реализация брокера событий сообщений через Redis Pub/Sub."""

    def __init__(self, redis_config: RedisConfig) -> None:
        """Создаёт брокер с настройками подключения к Redis."""
        self._redis_url = f"redis://{redis_config.host}:{redis_config.port}/{redis_config.db}"
        self._channel = redis_config.messages_channel
        self._publisher: redis.Redis | None = None
        self._pubsub: redis.client.PubSub | None = None
        self._listener_task: asyncio.Task[None] | None = None
        self._start_lock = asyncio.Lock()
        self._handler: MessageEventHandler | None = None

    async def start(self, handler: MessageEventHandler) -> None:
        """Запускает единственный Redis listener для текущего процесса."""
        async with self._start_lock:
            if self._listener_task is not None:
                return

            self._handler = handler
            if self._publisher is None:
                self._publisher = redis.from_url(self._redis_url, decode_responses=True)
            self._pubsub = self._publisher.pubsub()
            await self._pubsub.subscribe(self._channel)
            self._listener_task = asyncio.create_task(self._listen())

    async def _listen(self) -> None:
        """Читает события из Redis и передаёт их локальному обработчику."""
        if self._pubsub is None or self._handler is None:
            return

        while True:
            message = await self._pubsub.get_message(ignore_subscribe_messages=True, timeout=1.0)
            if message is None:
                continue

            data = json.loads(cast(str, message["data"]))
            conversation_id = int(data["conversation_id"])
            event = cast(MessageEvent, data["event"])
            await self._handler(conversation_id, event)

    async def publish(self, conversation_id: int, event: MessageEvent) -> None:
        """Публикует сериализуемое событие в общий Redis-канал."""
        if self._publisher is None:
            async with self._start_lock:
                if self._publisher is None:
                    self._publisher = redis.from_url(self._redis_url, decode_responses=True)
        if self._publisher is None:
            return

        await self._publisher.publish(
            self._channel,
            json.dumps(
                {"conversation_id": conversation_id, "event": event},
                ensure_ascii=False,
            ),
        )

    async def close(self) -> None:
        """Останавливает listener и закрывает Redis-соединения."""
        if self._listener_task is not None:
            self._listener_task.cancel()
            await asyncio.gather(self._listener_task, return_exceptions=True)
            self._listener_task = None

        if self._pubsub is not None:
            await cast(Any, self._pubsub).aclose()
            self._pubsub = None
        if self._publisher is not None:
            await self._publisher.aclose()
            self._publisher = None
