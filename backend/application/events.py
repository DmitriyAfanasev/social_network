from dataclasses import dataclass
from datetime import UTC, datetime
from uuid import uuid4


type JsonPrimitive = str | int | float | bool | None
type JsonValue = JsonPrimitive | list[JsonValue] | dict[str, JsonValue]
type JsonPayload = dict[str, JsonValue]


@dataclass(frozen=True, slots=True)
class ApplicationEvent:
    event_id: str
    occurred_at: datetime

    @classmethod
    def new(cls) -> "ApplicationEvent":
        return cls(
            event_id=str(uuid4()),
            occurred_at=datetime.now(UTC),
        )


@dataclass(frozen=True, slots=True)
class UserRegisteredEvent(ApplicationEvent):
    user_id: int
    email: str


@dataclass(frozen=True, slots=True)
class PostLikedEvent(ApplicationEvent):
    post_id: int
    user_id: int


@dataclass(frozen=True)
class IntegrationEvent:
    event_type: str
    payload: JsonPayload
