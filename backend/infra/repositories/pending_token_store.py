import logging
from typing import Any, cast

import redis.asyncio as redis

from backend.application.ports.token_store import PendingTokenStore
from backend.infra.config import RedisConfig


logger = logging.getLogger(__name__)


class RedisPendingTokenStore(PendingTokenStore):
    def __init__(
        self,
        redis_config: RedisConfig,
        redis_instance: Any = None,
    ) -> None:
        self.redis_url = f"redis://{redis_config.host}:{redis_config.port}/{redis_config.db}"
        self._redis = redis_instance

    async def _client(self) -> Any:
        if self._redis is None:
            self._redis = redis.from_url(
                self.redis_url,
                decode_responses=True,
            )
        return self._redis

    async def save_registration_confirmation_token(
        self,
        token: str,
        email: str,
        expires_sec: int = 1800,
    ) -> bool:
        return await self._save_token(token, email, expires_sec)

    async def save_password_reset_token(
        self,
        token: str,
        email: str,
        expires_sec: int = 600,
    ) -> bool:
        return await self._save_token(token, email, expires_sec)

    async def get_email_by_token(self, token: str) -> str | None:
        try:
            client = await self._client()
            return cast(str | None, await client.get(token))
        except Exception as e:
            logger.error("Pending token store read failed: %s", e, exc_info=True)
            raise RuntimeError("Pending token store read failed") from e

    async def delete_token(self, token: str) -> None:
        try:
            client = await self._client()
            await client.delete(token)
        except Exception as e:
            logger.error("Pending token store delete failed: %s", e, exc_info=True)
            raise RuntimeError("Pending token store delete failed") from e

    async def _save_token(self, token: str, email: str, expires_sec: int) -> bool:
        try:
            client = await self._client()
            await client.setex(token, expires_sec, email)
            return True
        except Exception as e:
            logger.error("Pending token store write failed: %s", e, exc_info=True)
            return False
