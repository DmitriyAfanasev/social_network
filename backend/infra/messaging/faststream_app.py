import asyncio

from faststream import AckPolicy, FastStream
from faststream.kafka import KafkaBroker

from backend.application.event_types import (
    PASSWORD_RESET_REQUESTED_EVENT,
    PROFILE_PHOTO_DELETED_EVENT,
    REGISTRATION_CONFIRMATION_REQUESTED_EVENT,
)
from backend.infra.config import settings
from backend.infra.messaging.outbox_publisher_worker import OutboxPublisherWorker
from backend.infra.messaging.schemas import (
    EmailNotificationPayload,
    ProfilePhotoDeletedPayload,
)
from backend.infra.notifications.email_service import create_email_service
from backend.infra.storage.factory import create_file_upload_service


broker = KafkaBroker(
    settings.event_bus.bootstrap_servers,
    acks=settings.event_bus.producer_acks,
    enable_idempotence=settings.event_bus.producer_enable_idempotence,
)
"""Kafka broker для FastStream.

FastAPI-приложение не использует этот объект напрямую. Он живёт только в
отдельном worker-процессе, который запускается через `./start-events.sh`.
"""

app = FastStream(broker)
"""FastStream application.

В одном процессе здесь собраны две роли:
1. outbox publisher читает pending-события из PostgreSQL и публикует их в Kafka;
2. subscribers получают неаналитические события из Kafka и выполняют внешние side effects.
"""

outbox_publisher = OutboxPublisherWorker(broker)
file_upload_service = create_file_upload_service(settings.file_storage)
email_service = create_email_service()


@app.after_startup
async def start_outbox_publisher() -> None:
    """Запустить фоновую задачу, которая переносит события из БД в Kafka."""
    outbox_publisher.start()


@app.after_shutdown
async def stop_outbox_publisher() -> None:
    """Остановить outbox publisher перед завершением FastStream worker-а."""
    await outbox_publisher.stop()


@broker.subscriber(
    PROFILE_PHOTO_DELETED_EVENT,
    group_id=settings.event_bus.consumer_group,
    ack_policy=AckPolicy.ACK,
)
async def delete_profile_photo_file(payload: ProfilePhotoDeletedPayload) -> None:
    """Удалить физический файл после события `profile_photo.deleted`.

    Если удаление завершится исключением, offset не будет подтверждён, и Kafka
    сможет доставить сообщение повторно. Если файла уже нет, storage adapter
    считает это успешной идемпотентной операцией.
    """
    deleted = await file_upload_service.delete(file_url=payload.file_url())
    if not deleted:
        raise RuntimeError(f"Unsupported profile photo file URL: {payload.file_url()}")


@broker.subscriber(
    REGISTRATION_CONFIRMATION_REQUESTED_EVENT,
    group_id=settings.event_bus.consumer_group,
    ack_policy=AckPolicy.ACK,
)
async def send_registration_confirmation_email(payload: EmailNotificationPayload) -> None:
    """Отправить письмо подтверждения регистрации по email-событию."""
    await send_token_email(
        payload=payload,
        name_endpoint="confirm-registration",
        name_message="initial_message",
    )


@broker.subscriber(
    PASSWORD_RESET_REQUESTED_EVENT,
    group_id=settings.event_bus.consumer_group,
    ack_policy=AckPolicy.ACK,
)
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
