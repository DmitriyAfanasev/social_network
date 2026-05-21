from datetime import datetime

from pydantic import BaseModel, Field

from backend.application.events import JsonPayload


class AnalyticsEventPayload(BaseModel):
    """Payload событий, которые пишутся в ClickHouse raw analytics table."""

    event_id: str = Field(description="Уникальный ID события для анализа дублей.")
    occurred_at: datetime = Field(description="UTC-время возникновения события в application layer.")
    user_id: int | None = Field(default=None, description="ID пользователя-инициатора, если он известен.")
    entity_type: str = Field(description="Тип сущности: user, post, comment, profile_photo и т.д.")
    entity_id: int | None = Field(default=None, description="ID сущности, к которой относится событие.")
    data: JsonPayload = Field(default_factory=dict, description="Дополнительный JSON payload события.")


class ProfilePhotoDeletedPayload(AnalyticsEventPayload):
    """Payload события `profile_photo.deleted`.

    Сообщение приходит в consumer после того, как фото уже удалено из альбома
    в базе данных. Consumer использует `file_url`, чтобы удалить физический файл
    из текущего storage adapter: локального диска или S3.
    """

    def file_url(self) -> str:
        file_url = self.data.get("file_url")
        if not isinstance(file_url, str) or not file_url:
            raise ValueError("profile_photo.deleted payload must contain data.file_url")
        return file_url


class EmailNotificationPayload(BaseModel):
    """Payload email-событий для подтверждения почты и сброса пароля."""

    email: str = Field(description="Email получателя письма.")
    token: str = Field(description="Одноразовый токен, который будет добавлен в ссылку.")
    base_url: str = Field(description="Базовый URL фронтенда для сборки ссылки в письме.")
