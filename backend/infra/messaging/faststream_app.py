import asyncio

from faststream import AckPolicy, FastStream
from faststream.rabbit import RabbitBroker

from backend.application.event_types import (
    COMMENT_CREATED_EVENT,
    PASSWORD_RESET_REQUESTED_EVENT,
    POST_CREATED_EVENT,
    POST_LIKE_TOGGLED_EVENT,
    PROFILE_PHOTO_DELETED_EVENT,
    REGISTRATION_CONFIRMATION_REQUESTED_EVENT,
    USER_REGISTERED_EVENT,
)
from backend.infra.analytics.clickhouse_client import ClickHouseAnalyticsClient
from backend.infra.config import settings
from backend.infra.messaging.outbox_publisher_worker import OutboxPublisherWorker
from backend.infra.messaging.schemas import (
    AnalyticsEventPayload,
    EmailNotificationPayload,
    ProfilePhotoDeletedPayload,
)
from backend.infra.notifications.email_service import create_email_service
from backend.infra.storage.factory import create_file_upload_service


broker = RabbitBroker(str(settings.event_bus.broker_url))
"""RabbitMQ broker для FastStream.

FastAPI-приложение не использует этот объект напрямую. Он живёт только в
отдельном worker-процессе, который запускается через `./start-events.sh`.
"""

app = FastStream(broker)
"""FastStream application.

В одном процессе здесь собраны две роли:
1. outbox publisher читает pending-события из PostgreSQL и публикует их в RabbitMQ;
2. subscribers получают события из RabbitMQ и выполняют внешние side effects.
"""

outbox_publisher = OutboxPublisherWorker(broker)
file_upload_service = create_file_upload_service(settings.file_storage)
email_service = create_email_service()
clickhouse_client: ClickHouseAnalyticsClient | None = None


@app.after_startup
async def start_outbox_publisher() -> None:
    """Запустить фоновую задачу, которая переносит события из БД в RabbitMQ."""
    global clickhouse_client

    clickhouse_client = await ClickHouseAnalyticsClient.create(settings.clickhouse)
    await clickhouse_client.init_schema()
    outbox_publisher.start()


@app.after_shutdown
async def stop_outbox_publisher() -> None:
    """Остановить outbox publisher перед завершением FastStream worker-а."""
    await outbox_publisher.stop()
    if clickhouse_client is not None:
        await clickhouse_client.close()


@broker.subscriber(PROFILE_PHOTO_DELETED_EVENT, ack_policy=AckPolicy.NACK_ON_ERROR)
async def delete_profile_photo_file(payload: ProfilePhotoDeletedPayload) -> None:
    """Удалить физический файл после события `profile_photo.deleted`.

    Если удаление завершится исключением, FastStream сделает NACK, и RabbitMQ
    сможет доставить сообщение повторно. Если файла уже нет, storage adapter
    считает это успешной идемпотентной операцией.
    """
    await write_analytics_event(PROFILE_PHOTO_DELETED_EVENT, payload)
    deleted = await file_upload_service.delete(file_url=payload.file_url())
    if not deleted:
        raise RuntimeError(f"Unsupported profile photo file URL: {payload.file_url()}")


@broker.subscriber(USER_REGISTERED_EVENT, ack_policy=AckPolicy.NACK_ON_ERROR)
async def write_user_registered_analytics(payload: AnalyticsEventPayload) -> None:
    """Записать регистрацию пользователя в ClickHouse."""
    await write_analytics_event(USER_REGISTERED_EVENT, payload)


@broker.subscriber(POST_CREATED_EVENT, ack_policy=AckPolicy.NACK_ON_ERROR)
async def write_post_created_analytics(payload: AnalyticsEventPayload) -> None:
    """Записать создание поста в ClickHouse."""
    await write_analytics_event(POST_CREATED_EVENT, payload)


@broker.subscriber(POST_LIKE_TOGGLED_EVENT, ack_policy=AckPolicy.NACK_ON_ERROR)
async def write_post_like_toggled_analytics(payload: AnalyticsEventPayload) -> None:
    """Записать добавление или снятие лайка в ClickHouse."""
    await write_analytics_event(POST_LIKE_TOGGLED_EVENT, payload)


@broker.subscriber(COMMENT_CREATED_EVENT, ack_policy=AckPolicy.NACK_ON_ERROR)
async def write_comment_created_analytics(payload: AnalyticsEventPayload) -> None:
    """Записать создание комментария в ClickHouse."""
    await write_analytics_event(COMMENT_CREATED_EVENT, payload)


@broker.subscriber(REGISTRATION_CONFIRMATION_REQUESTED_EVENT, ack_policy=AckPolicy.NACK_ON_ERROR)
async def send_registration_confirmation_email(payload: EmailNotificationPayload) -> None:
    """Отправить письмо подтверждения регистрации по email-событию."""
    await send_token_email(
        payload=payload,
        name_endpoint="confirm-registration",
        name_message="initial_message",
    )


@broker.subscriber(PASSWORD_RESET_REQUESTED_EVENT, ack_policy=AckPolicy.NACK_ON_ERROR)
async def send_password_reset_email(payload: EmailNotificationPayload) -> None:
    """Отправить письмо сброса пароля по email-событию."""
    await send_token_email(
        payload=payload,
        name_endpoint="reset_password",
        name_message="message_reset_password",
    )


async def send_token_email(
    *,
    payload: EmailNotificationPayload,
    name_endpoint: str,
    name_message: str,
) -> None:
    """Собрать token-ссылку, HTML-письмо и отправить его через SMTP.

    SMTP-клиент синхронный, поэтому отправка уходит в `asyncio.to_thread`, чтобы
    не блокировать event loop FastStream worker-а.
    """
    link = email_service.build_confirmation_link(
        name_endpoint,
        payload.base_url,
        payload.token,
    )
    message = email_service.compose_email(
        name_message,
        payload.email,
        link,
    )
    sent = await asyncio.to_thread(email_service.send_email, message, payload.email)
    if not sent:
        raise RuntimeError(f"Email was not sent to {payload.email}")


async def write_analytics_event(event_type: str, payload: AnalyticsEventPayload) -> None:
    """Записать raw analytics event в ClickHouse.

    Если ClickHouse недоступен или insert падает, consumer бросит исключение,
    FastStream сделает NACK, а RabbitMQ повторно доставит сообщение.
    """
    if clickhouse_client is None:
        raise RuntimeError("ClickHouse analytics client is not initialized")

    await clickhouse_client.insert_event(event_type=event_type, payload=payload)
