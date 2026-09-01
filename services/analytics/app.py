import asyncio
import logging
from collections.abc import Awaitable, Callable
from datetime import UTC, datetime

from faststream import AckPolicy, FastStream
from faststream.kafka import KafkaBroker
from prometheus_client import Counter, start_http_server

from backend.application.event_types import ANALYTICS_EVENT_TYPES
from backend.infra.analytics.clickhouse_client import ClickHouseAnalyticsClient
from backend.infra.config import settings
from backend.infra.messaging.schemas import AnalyticsEventPayload


broker = KafkaBroker(
    settings.event_bus.bootstrap_servers,
    acks=settings.event_bus.producer_acks,
    enable_idempotence=settings.event_bus.producer_enable_idempotence,
)
app = FastStream(broker)
clickhouse_client: ClickHouseAnalyticsClient | None = None
logger = logging.getLogger(__name__)

events_processed = Counter(
    "analytics_events_processed_total",
    "Successfully persisted analytics events.",
    ("event_type",),
)
events_failed = Counter(
    "analytics_events_failed_total",
    "Failed analytics processing attempts.",
    ("event_type",),
)
events_retried = Counter(
    "analytics_events_retried_total",
    "Analytics processing retries.",
    ("event_type",),
)
events_sent_to_dlq = Counter(
    "analytics_dlq_events_total",
    "Analytics events sent to Kafka DLQ.",
    ("event_type",),
)


@app.after_startup
async def start_analytics_consumer() -> None:
    """Initialize ClickHouse once for the standalone analytics process."""
    global clickhouse_client
    start_http_server(settings.event_bus.analytics_metrics_port)
    clickhouse_client = await ClickHouseAnalyticsClient.create(settings.clickhouse)
    await clickhouse_client.init_schema()


@app.after_shutdown
async def stop_analytics_consumer() -> None:
    """Release ClickHouse resources when the consumer stops."""
    if clickhouse_client is not None:
        await clickhouse_client.close()


async def persist_analytics_event(event_type: str, payload: AnalyticsEventPayload) -> None:
    """Persist one validated Kafka event in the raw analytics table."""
    if clickhouse_client is None:
        raise RuntimeError("ClickHouse analytics client is not initialized")
    await clickhouse_client.insert_event(
        event_type=event_type,
        payload=payload,
    )


async def process_analytics_event(event_type: str, payload: AnalyticsEventPayload) -> None:
    """Persist an event, retry transient failures, then isolate poison events."""
    for attempt in range(1, settings.event_bus.analytics_max_attempts + 1):
        try:
            await persist_analytics_event(event_type, payload)
        except Exception as error:
            events_failed.labels(event_type).inc()
            if attempt == settings.event_bus.analytics_max_attempts:
                await broker.publish(
                    {
                        "event_type": event_type,
                        "payload": payload.model_dump(mode="json"),
                        "error": str(error),
                        "failed_at": datetime.now(UTC).isoformat(),
                        "attempts": attempt,
                    },
                    topic=settings.event_bus.analytics_dlq_topic,
                )
                events_sent_to_dlq.labels(event_type).inc()
                logger.exception(
                    "Analytics event sent to DLQ: event_type=%s event_id=%s",
                    event_type,
                    payload.event_id,
                )
                return

            events_retried.labels(event_type).inc()
            await asyncio.sleep(
                settings.event_bus.analytics_retry_base_seconds * 2 ** (attempt - 1)
            )
        else:
            events_processed.labels(event_type).inc()
            return


def create_event_handler(
    event_type: str,
) -> Callable[[AnalyticsEventPayload], Awaitable[None]]:
    """Create a topic-specific handler while keeping one persistence strategy."""

    async def handler(payload: AnalyticsEventPayload) -> None:
        await process_analytics_event(event_type, payload)

    return handler


for topic in ANALYTICS_EVENT_TYPES:
    broker.subscriber(
        topic,
        group_id=settings.event_bus.analytics_consumer_group,
        ack_policy=AckPolicy.ACK,
    )(create_event_handler(topic))
