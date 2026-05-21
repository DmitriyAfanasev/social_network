from dataclasses import dataclass
from datetime import date


@dataclass(kw_only=True)
class Profile:
    first_name: str | None = None
    last_name: str | None = None
    middle_name: str | None = None
    birth_date: date | None = None
    gender: str | None = None
    phone_number: str | None = None
    country: str | None = None
    city: str | None = None
    street: str | None = None
    bio: str | None = None
    avatar: str | None = None

    @property
    def full_name(self) -> str | None:
        if self.first_name and self.last_name:
            return f"{self.first_name} {self.last_name}"
        return None
