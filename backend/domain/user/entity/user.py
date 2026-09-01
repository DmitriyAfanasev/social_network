from dataclasses import dataclass, field
from datetime import UTC, datetime

from backend.domain.shared.base_entity import BaseEntity
from backend.domain.user.entity.profile import Profile


@dataclass(kw_only=True)
class User(BaseEntity):
    username: str
    email: str
    hashed_password: str
    is_active: bool = True
    is_superuser: bool = False
    last_seen_at: datetime | None = None
    created_at: datetime = field(default_factory=lambda: datetime.now(UTC))
    profile: Profile | None = None

    @property
    def full_name(self) -> str:
        if self.profile and self.profile.full_name:
            return self.profile.full_name
        return self.username
