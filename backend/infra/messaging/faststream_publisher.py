from faststream.kafka import KafkaBroker

from backend.application.events import JsonPayload
from backend.application.ports.event_publisher import EventPublisher


class FastStreamEventPublisher(EventPublisher):
    """Публикует integration events в Kafka через FastStream broker.

    Outbox use case ничего не знает про Kafka/FastStream и работает только
    с портом `EventPublisher`. Этот adapter связывает application-слой с
    конкретной messaging-инфраструктурой.
    """

    def __init__(self, broker: KafkaBroker) -> None:
        self.broker = broker

    async def publish(self, *, event_type: str, payload: JsonPayload) -> None:
        """Опубликовать payload в topic, равный типу события."""
        await self.broker.publish(payload, topic=event_type)
