from datetime import date
from typing import TYPE_CHECKING

from sqlalchemy import Boolean, Date, ForeignKey, String, Text
from sqlalchemy.orm import Mapped, mapped_column, relationship

from .base import Base


if TYPE_CHECKING:
    from .user import User


class Profile(Base):
    """Профиль пользователя с личной информацией."""

    user_id: Mapped[int] = mapped_column(
        ForeignKey("users.id"),
        unique=True,
        nullable=False,
    )
    user: Mapped["User"] = relationship("User", back_populates="profile")

    first_name: Mapped[str | None] = mapped_column(String(50))
    last_name: Mapped[str | None] = mapped_column(String(50))
    middle_name: Mapped[str | None] = mapped_column(String(50))

    birth_date: Mapped[date | None] = mapped_column(Date)
    gender: Mapped[str | None]

    phone_number: Mapped[str | None] = mapped_column(String(20))
    country: Mapped[str | None] = mapped_column(String(50))
    city: Mapped[str | None] = mapped_column(String(50))
    street: Mapped[str | None] = mapped_column(String(100))

    bio: Mapped[str | None] = mapped_column(Text())
    avatar: Mapped[str] = mapped_column(
        Text(),
        default="/media/default-avatar",
        server_default="/media/default-avatar",
    )
    profile_visibility: Mapped[str] = mapped_column(String(24), nullable=False, default="everyone", server_default="everyone")
    friend_request_policy: Mapped[str] = mapped_column(String(24), nullable=False, default="everyone", server_default="everyone")
    message_policy: Mapped[str] = mapped_column(String(24), nullable=False, default="everyone", server_default="everyone")
    show_email: Mapped[bool] = mapped_column(Boolean, nullable=False, default=False, server_default="false")
    show_phone: Mapped[bool] = mapped_column(Boolean, nullable=False, default=False, server_default="false")
    show_birth_date: Mapped[bool] = mapped_column(Boolean, nullable=False, default=True, server_default="true")
    show_friends: Mapped[bool] = mapped_column(Boolean, nullable=False, default=True, server_default="true")
    show_posts: Mapped[bool] = mapped_column(Boolean, nullable=False, default=True, server_default="true")

    @property
    def full_name(self) -> str:
        """Возвращает полное имя пользователя."""
        return (
            f"{self.first_name} {self.last_name}"
            if (self.first_name and self.last_name)
            else self.user.username
        )
