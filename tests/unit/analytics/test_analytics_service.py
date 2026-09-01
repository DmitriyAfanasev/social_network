from unittest.mock import AsyncMock

import pytest

from backend.application.event_types import FRIEND_REQUESTED_EVENT
from backend.infra.messaging.schemas import AnalyticsEventPayload
from services.analytics import app as analytics_app


@pytest.mark.asyncio
async def test_analytics_handler_persists_event_with_topic_name() -> None:
    client = AsyncMock()
    analytics_app.clickhouse_client = client
    payload = AnalyticsEventPayload(
        event_id="event-1",
        occurred_at="2026-09-01T10:00:00Z",
        user_id=1,
        entity_type="friendship",
        entity_id=2,
        data={"action": "requested"},
    )

    await analytics_app.create_event_handler(FRIEND_REQUESTED_EVENT)(payload)

    client.insert_event.assert_awaited_once_with(
        event_type=FRIEND_REQUESTED_EVENT,
        payload=payload,
    )
    analytics_app.clickhouse_client = None


@pytest.mark.asyncio
async def test_failed_event_is_sent_to_dlq_after_attempt_limit(monkeypatch: pytest.MonkeyPatch) -> None:
    client = AsyncMock()
    client.insert_event.side_effect = RuntimeError("ClickHouse is unavailable")
    publish = AsyncMock()
    monkeypatch.setattr(analytics_app, "clickhouse_client", client)
    monkeypatch.setattr(analytics_app.broker, "publish", publish)
    monkeypatch.setattr(analytics_app.settings.event_bus, "analytics_max_attempts", 1)

    payload = AnalyticsEventPayload(
        event_id="event-2",
        occurred_at="2026-09-01T10:00:00Z",
        user_id=1,
        entity_type="post",
        entity_id=2,
    )

    await analytics_app.create_event_handler("post.created")(payload)

    publish.assert_awaited_once()
    assert publish.await_args.kwargs["topic"] == analytics_app.settings.event_bus.analytics_dlq_topic
    assert publish.await_args.args[0]["event_type"] == "post.created"
