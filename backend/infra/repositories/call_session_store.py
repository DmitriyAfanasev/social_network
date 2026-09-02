import json
from typing import TypedDict, cast

import redis.asyncio as redis
from redis.exceptions import RedisError

from backend.application.exceptions import CallSessionStoreError
from backend.application.ports.call_session_store import CallSessionStore
from backend.domain.call import CallSession, CallStatus, CallType
from backend.infra.config import RedisConfig


class _CallSessionPayload(TypedDict):
    call_id: str
    caller_id: int
    callee_id: int
    call_type: str
    status: str


class RedisCallSessionStore(CallSessionStore):
    """Shared call sessions for all application workers with an expiry."""

    SESSION_TTL_SECONDS = 120

    def __init__(self, config: RedisConfig, redis_instance: redis.Redis | None = None) -> None:
        self._redis_url = f"redis://{config.host}:{config.port}/{config.db}"
        self._redis = redis_instance

    async def _client(self) -> redis.Redis:
        if self._redis is None:
            self._redis = redis.from_url(self._redis_url, decode_responses=True)
        return self._redis

    async def save(self, session: CallSession) -> None:
        payload = json.dumps(
            {
                "call_id": session.call_id,
                "caller_id": session.caller_id,
                "callee_id": session.callee_id,
                "call_type": session.call_type.value,
                "status": session.status.value,
            }
        )
        try:
            await (await self._client()).setex(
                self._key(session.call_id), self.SESSION_TTL_SECONDS, payload
            )
        except (OSError, RedisError) as error:
            raise CallSessionStoreError from error

    async def get(self, call_id: str) -> CallSession | None:
        try:
            payload = await (await self._client()).get(self._key(call_id))
        except (OSError, RedisError) as error:
            raise CallSessionStoreError from error
        if payload is None:
            return None
        data = cast(_CallSessionPayload, json.loads(payload))
        return CallSession(
            call_id=str(data["call_id"]),
            caller_id=int(data["caller_id"]),
            callee_id=int(data["callee_id"]),
            call_type=CallType(str(data["call_type"])),
            status=CallStatus(str(data["status"])),
        )

    async def delete(self, call_id: str) -> None:
        try:
            await (await self._client()).delete(self._key(call_id))
        except (OSError, RedisError) as error:
            raise CallSessionStoreError from error

    @staticmethod
    def _key(call_id: str) -> str:
        return f"calls:session:{call_id}"
