import importlib
import json
from collections.abc import Mapping, Sequence
from datetime import datetime
from typing import Protocol, cast

from backend.application.results import AnalyticsSummaryResult
from backend.infra.config import ClickHouseConfig
from backend.infra.messaging.schemas import AnalyticsEventPayload


type ClickHouseValue = str | int | float | bool | datetime | None
type ClickHouseRow = Sequence[ClickHouseValue]


class ClickHouseQueryResult(Protocol):
    result_rows: Sequence[ClickHouseRow]


class ClickHouseAsyncClient(Protocol):
    async def command(self, cmd: str) -> ClickHouseValue:
        pass

    async def insert(
        self,
        table: str,
        data: Sequence[ClickHouseRow],
        column_names: Sequence[str],
    ) -> None:
        pass

    async def query(
        self,
        query: str,
        parameters: Mapping[str, ClickHouseValue] | None = None,
    ) -> ClickHouseQueryResult:
        pass

    async def close(self) -> None:
        pass


class ClickHouseAnalyticsClient:
    """Небольшой adapter вокруг clickhouse-connect для analytics use cases.

    Импорт `clickhouse_connect` сделан ленивым в `create`, чтобы обычный импорт
    FastAPI/FastStream модулей не падал до установки зависимости.
    """

    def __init__(
        self,
        *,
        client: ClickHouseAsyncClient,
        database: str,
    ) -> None:
        self.client = client
        self.database = _safe_identifier(database)
        self.table_name = f"`{self.database}`.`analytics_events`"

    @classmethod
    async def create(cls, config: ClickHouseConfig) -> "ClickHouseAnalyticsClient":
        clickhouse_connect = importlib.import_module("clickhouse_connect")
        client = cast(
            ClickHouseAsyncClient,
            await clickhouse_connect.get_async_client(
                host=config.host,
                port=config.port,
                username=config.username,
                password=config.password.get_secret_value(),
                database="default",
                secure=config.secure,
            ),
        )
        return cls(client=client, database=config.database)

    async def init_schema(self) -> None:
        """Создать базу и raw events table, если они ещё не существуют."""
        await self.client.command(f"CREATE DATABASE IF NOT EXISTS `{self.database}`")
        await self.client.command(
            f"""
            CREATE TABLE IF NOT EXISTS {self.table_name}
            (
                event_id String,
                event_type LowCardinality(String),
                occurred_at DateTime64(3, 'UTC'),
                user_id Nullable(UInt64),
                entity_type LowCardinality(String),
                entity_id Nullable(UInt64),
                payload_json String
            )
            ENGINE = ReplacingMergeTree
            PARTITION BY toYYYYMM(occurred_at)
            ORDER BY (event_type, occurred_at, entity_type, event_id)
            """
        )

    async def insert_event(self, *, event_type: str, payload: AnalyticsEventPayload) -> None:
        """Записать одно analytics-событие в raw ClickHouse table."""
        await self.client.insert(
            self.table_name,
            [
                (
                    payload.event_id,
                    event_type,
                    payload.occurred_at,
                    payload.user_id,
                    payload.entity_type,
                    payload.entity_id,
                    json.dumps(payload.data, ensure_ascii=False, sort_keys=True),
                )
            ],
            column_names=[
                "event_id",
                "event_type",
                "occurred_at",
                "user_id",
                "entity_type",
                "entity_id",
                "payload_json",
            ],
        )

    async def get_summary(self) -> AnalyticsSummaryResult:
        """Прочитать простой summary по всем накопленным raw событиям."""
        result = await self.client.query(
            f"""
            SELECT
                count(),
                uniqExact(user_id),
                countIf(event_type = 'user.registered'),
                countIf(event_type = 'post.created'),
                countIf(event_type = 'comment.created'),
                countIf(event_type = 'post.like_toggled' AND JSONExtractString(payload_json, 'action') = 'added'),
                countIf(event_type = 'post.like_toggled' AND JSONExtractString(payload_json, 'action') = 'removed'),
                countIf(event_type = 'profile_photo.deleted')
            FROM {self.table_name} FINAL
            """
        )
        row = result.result_rows[0] if result.result_rows else (0, 0, 0, 0, 0, 0, 0, 0)
        return AnalyticsSummaryResult(
            events_count=_to_int(row[0]),
            unique_users=_to_int(row[1]),
            users_registered=_to_int(row[2]),
            posts_created=_to_int(row[3]),
            comments_created=_to_int(row[4]),
            likes_added=_to_int(row[5]),
            likes_removed=_to_int(row[6]),
            photos_deleted=_to_int(row[7]),
        )

    async def close(self) -> None:
        await self.client.close()


def _safe_identifier(value: str) -> str:
    if not value or not all(part.isidentifier() for part in value.split(".")):
        raise ValueError("ClickHouse identifier must contain only valid identifier parts")
    return value


def _to_int(value: ClickHouseValue) -> int:
    if value is None:
        return 0
    if isinstance(value, datetime):
        raise TypeError("ClickHouse summary value must be numeric")
    return int(value)
