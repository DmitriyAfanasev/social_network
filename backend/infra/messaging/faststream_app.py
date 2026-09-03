import asyncio
from typing import Any

from faststream import AckPolicy, FastStream
from faststream.kafka import KafkaBroker
from sqlalchemy.ext.asyncio import AsyncEngine, AsyncSession, async_sessionmaker, create_async_engine

from backend.application.event_types import (
    PASSWORD_RESET_REQUESTED_EVENT,
    PROFILE_PHOTO_DELETED_EVENT,
    REGISTRATION_CONFIRMATION_REQUESTED_EVENT,
)
from backend.application.ports.video_repository import VideoRenditionInput
from backend.infra.config import settings
from backend.infra.messaging.outbox_publisher_worker import OutboxPublisherWorker
from backend.infra.messaging.schemas import (
    EmailNotificationPayload,
    ProfilePhotoDeletedPayload,
)
from backend.infra.notifications.email_service import create_email_service
from backend.infra.repositories.video_repository import SQLAlchemyVideoRepository
from backend.infra.storage.s3_file_upload_service import S3FileUploadService
from backend.infra.transactions.sqlalchemy import SQLAlchemyTransactionManager


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
file_upload_service = S3FileUploadService(
    bucket_name=settings.file_storage.s3_bucket_name,
    max_size_bytes=settings.file_storage.max_file_size_bytes,
    region_name=settings.file_storage.s3_region_name,
    endpoint_url=settings.file_storage.s3_endpoint_url,
    access_key_id=settings.file_storage.s3_access_key_id,
    secret_access_key=settings.file_storage.s3_secret_access_key,
    public_base_url=settings.file_storage.s3_public_base_url,
    key_prefix=settings.file_storage.s3_key_prefix,
)
email_service = create_email_service()
video_processing_engine: AsyncEngine | None = None
video_session_factory: async_sessionmaker[AsyncSession] | None = None


@app.after_startup
async def start_outbox_publisher() -> None:
    """Запустить фоновую задачу, которая переносит события из БД в Kafka."""
    global video_processing_engine, video_session_factory
    video_processing_engine = create_async_engine(
        url=str(settings.db.url),
        echo=settings.db.echo,
        echo_pool=settings.db.echo_pool,
        pool_size=settings.db.pool_size,
        max_overflow=settings.db.max_overflow,
        pool_pre_ping=True,
    )
    video_session_factory = async_sessionmaker(
        bind=video_processing_engine,
        autoflush=False,
        autocommit=False,
        expire_on_commit=False,
    )
    outbox_publisher.start()


@app.after_shutdown
async def stop_outbox_publisher() -> None:
    """Остановить outbox publisher перед завершением FastStream worker-а."""
    global video_processing_engine, video_session_factory
    await outbox_publisher.stop()
    if video_processing_engine is not None:
        await video_processing_engine.dispose()
    video_processing_engine = None
    video_session_factory = None


@broker.subscriber(
    "video.transcode.completed",
    group_id=settings.event_bus.consumer_group,
    ack_policy=AckPolicy.ACK,
)
async def complete_video_transcoding(payload: dict[str, Any]) -> None:
    """Сохранить результат Go-транскодера в PostgreSQL."""
    if video_session_factory is None:
        raise RuntimeError("Video processing database is not initialized")
    video_id = int(payload["video_id"])
    status = str(payload.get("status", "failed"))
    error_message = str(payload["error_message"]) if payload.get("error_message") else None
    raw_renditions = payload.get("renditions", [])
    if not isinstance(raw_renditions, list):
        raw_renditions = []
    renditions = tuple(
        VideoRenditionInput(
            height=int(item["height"]),
            object_key=str(item["object_key"]),
            content_type=str(item.get("content_type", "video/mp4")),
            size=int(item.get("size", 0)),
            width=int(item["width"]) if item.get("width") is not None else None,
            duration=float(item["duration"]) if item.get("duration") is not None else None,
        )
        for item in raw_renditions
        if isinstance(item, dict) and "height" in item and "object_key" in item
    )
    duration = float(payload["duration"]) if payload.get("duration") is not None else None
    async with video_session_factory() as session:
        repository = SQLAlchemyVideoRepository(session)
        async with SQLAlchemyTransactionManager(session=session):
            updated = await repository.complete_transcoding(
                video_id=video_id,
                status=status,
                renditions=renditions,
                duration=duration,
                error_message=error_message,
            )
            if not updated:
                raise RuntimeError(f"Video {video_id} was not found")


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
