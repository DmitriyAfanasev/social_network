from faststream.rabbit import RabbitBroker

from backend.application.events import JsonPayload
from backend.application.ports.event_publisher import EventPublisher


class FastStreamEventPublisher(EventPublisher):
    """Публикует integration events в RabbitMQ через FastStream broker.

    Outbox use case ничего не знает про RabbitMQ/FastStream и работает только
    с портом `EventPublisher`. Этот adapter связывает application-слой с
    конкретной messaging-инфраструктурой.
    """

    def __init__(self, broker: RabbitBroker) -> None:
        self.broker = broker

    async def publish(self, *, event_type: str, payload: JsonPayload) -> None:
        """Опубликовать payload с routing key, равным типу события."""
        await self.broker.publish(payload, event_type)
