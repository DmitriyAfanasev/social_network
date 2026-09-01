from collections.abc import Awaitable, Callable

from faststream import AckPolicy, FastStream
from faststream.kafka import KafkaBroker

from backend.application.event_types import ANALYTICS_EVENT_TYPES
from backend.infra.analytics.clickhouse_client import ClickHouseAnalyticsClient
from backend.infra.config import settings
from backend.infra.messaging.schemas import AnalyticsEventPayload


broker = KafkaBroker(settings.event_bus.bootstrap_servers)
app = FastStream(broker)
clickhouse_client: ClickHouseAnalyticsClient | None = None


@app.after_startup
async def start_analytics_consumer() -> None:
    """Initialize ClickHouse once for the standalone analytics process."""
    global clickhouse_client
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


def create_event_handler(
    event_type: str,
) -> Callable[[AnalyticsEventPayload], Awaitable[None]]:
    """Create a topic-specific handler while keeping one persistence strategy."""

    async def handler(payload: AnalyticsEventPayload) -> None:
        await persist_analytics_event(event_type, payload)

    return handler


for topic in ANALYTICS_EVENT_TYPES:
    broker.subscriber(
        topic,
        group_id=settings.event_bus.analytics_consumer_group,
        ack_policy=AckPolicy.ACK,
    )(create_event_handler(topic))
