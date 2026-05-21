from dataclasses import dataclass, field
from datetime import UTC, datetime
from typing import Any


@dataclass(kw_only=True)
class BaseEntity:
    id: int | None = None
    created_at: datetime = field(default_factory=lambda: datetime.now(UTC))

    def to_dict(self) -> dict[str, Any]:
        return {
            k: v for k, v in self.__dict__.items() if not k.startswith("_")
        }

    def __str__(self) -> str:
        return f"{self.__class__.__name__}(id={self.id})"

    def __repr__(self) -> str:
        return str(self)
