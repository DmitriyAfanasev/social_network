import logging
from collections.abc import Callable
from pathlib import Path
from typing import Literal, TypeVar, cast

from fastapi.templating import Jinja2Templates
from pydantic import BaseModel, EmailStr, Field, PostgresDsn, SecretStr
from pydantic_settings import BaseSettings, SettingsConfigDict


BASE_DIR = Path(__file__).resolve().parent.parent.parent.parent

TEMPLATES_DIR = BASE_DIR / "frontend" / "templates"
TEMPLATES = Jinja2Templates(directory=str(TEMPLATES_DIR))

LOG_DEFAULT_FORMAT = (
    "[%(asctime)s.%(msecs)03d] %(module)10s:%(lineno)-3d %(levelname)-7s - %(message)s"
)

LOG_DATE_FORMAT = "%Y-%m-%d %H:%M:%S"

DEFAULT_AVATAR_OBJECT_KEY = "system/default-avatar.svg"
DEFAULT_AVATAR_URL = "/media/default-avatar"

type SMTPUser = EmailStr
type Algorithm = Literal[
    "HS256",
    "HS384",
    "HS512",
    "RS256",
    "RS384",
    "RS512",
    "ES256",
]
type FileStorageProvider = Literal["s3"]
type LogLevel = Literal[
    "debug",
    "info",
    "warning",
    "error",
    "critical",
]

T = TypeVar("T")


def _settings_factory(model: type[T]) -> Callable[[], T]:
    return cast(Callable[[], T], model)


class JwtConfig(BaseModel):
    secret_key: SecretStr
    algorithm: Algorithm
    access_token_expire_minutes: int = 30
    refresh_token_expire_days: int = 7


class LoggingConfig(BaseModel):
    log_level: LogLevel = "info"
    log_format: str = LOG_DEFAULT_FORMAT

    @property
    def log_level_value(self) -> int:
        """Функция для получения значения уровня логирования."""
        return logging.getLevelNamesMapping()[self.log_level.upper()]


class DatabaseConfig(BaseModel):
    url: PostgresDsn
    echo: bool = False
    echo_pool: bool = False
    pool_size: int = 50
    max_overflow: int = 10
    MODE: str = "TEST"


class RedisConfig(BaseModel):
    host: str
    port: int
    db: int
    messages_channel: str = "messages:events"


class EventBusConfig(BaseModel):
    bootstrap_servers: str = "localhost:9092"
    consumer_group: str = "general-project-events"
    analytics_consumer_group: str = "general-project-analytics"
    producer_acks: Literal[0, 1, "all"] = "all"
    producer_enable_idempotence: bool = True
    analytics_dlq_topic: str = "analytics.dlq"
    analytics_max_attempts: int = 3
    analytics_retry_base_seconds: float = 1.0
    analytics_metrics_port: int = 9101
    outbox_poll_interval_seconds: float = 1.0
    outbox_batch_size: int = 50


class ClickHouseConfig(BaseModel):
    host: str = "localhost"
    port: int = 8123
    username: str = "default"
    password: SecretStr = SecretStr("password")
    database: str = "default"
    secure: bool = False


class FrontendConfig(BaseModel):
    base_url: str = "http://127.0.0.1:5173"


class FileStorageConfig(BaseModel):
    provider: FileStorageProvider = "s3"
    max_file_size_mb: int = 5
    s3_bucket_name: str = "general-project"
    s3_region_name: str = "us-east-1"
    s3_endpoint_url: str = "http://localhost:9000"
    s3_access_key_id: SecretStr = SecretStr("minioadmin")
    s3_secret_access_key: SecretStr = SecretStr("minioadmin")
    s3_public_base_url: str | None = None
    s3_key_prefix: str = "media"

    @property
    def max_file_size_bytes(self) -> int:
        return self.max_file_size_mb * 1024 * 1024


class SMTPSettings(BaseModel):
    host: str
    port: int
    user: SMTPUser
    user_to_email: SMTPUser | None = None
    password: SecretStr
    use_tls: bool = True
    use_ssl: bool = False


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=(
            BASE_DIR / ".test.env",
            BASE_DIR / ".env.template",
            BASE_DIR / ".env",
            # порядок важен, т.к. pydantic_settings отдаёт приоритет
            # последнеиу файлу, и если последний файл это не продакшн,
            # .env то найстройки будут искаться в .env.template
        ),
        env_file_encoding="utf-8",
        case_sensitive=False,
        env_nested_delimiter="__",
        env_prefix="APP_CONFIG__",
        extra="ignore",
    )
    logging: LoggingConfig = Field(default_factory=LoggingConfig)
    jwt: JwtConfig = Field(default_factory=_settings_factory(JwtConfig))
    redis: RedisConfig = Field(default_factory=_settings_factory(RedisConfig))
    event_bus: EventBusConfig = Field(default_factory=EventBusConfig)
    clickhouse: ClickHouseConfig = Field(default_factory=ClickHouseConfig)
    frontend: FrontendConfig = Field(default_factory=FrontendConfig)
    file_storage: FileStorageConfig = Field(default_factory=FileStorageConfig)
    smtp: SMTPSettings = Field(default_factory=_settings_factory(SMTPSettings))
    db: DatabaseConfig = Field(default_factory=_settings_factory(DatabaseConfig))


CONVENTION = {
    "ix": "ix_%(column_0_label)s",
    "uq": "uq_%(table_name)s_%(column_0_name)s",
    "ck": "ck_%(table_name)s_%(constraint_name)s",
    "fk": "fk_%(table_name)s_%(column_0_name)s_%(referred_table_name)s",
    "pk": "pk_%(table_name)s",
}

settings = Settings()
