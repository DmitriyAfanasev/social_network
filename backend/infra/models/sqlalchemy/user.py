from typing import TYPE_CHECKING

from pydantic import EmailStr

from sqlalchemy import Boolean, CheckConstraint, String
from sqlalchemy.orm import Mapped, mapped_column, relationship

from .base import Base
from .mixins import TimestampsMixin


if TYPE_CHECKING:
    from .comment import Comment
    from .like import LikePost
    from .post import Post
    from .profile import Profile


class User(TimestampsMixin, Base):
    """Класс пользователя, которого определяет система."""

    __table_args__ = (
        CheckConstraint("username != ''", name="check_username_not_empty"),
        CheckConstraint("hashed_password != ''", name="check_password_not_empty"),
        CheckConstraint("email != ''", name="check_email_not_empty"),
    )
    id: Mapped[int] = mapped_column(unique=True, primary_key=True)
    username: Mapped[str] = mapped_column(String(20), unique=True, index=True)
    hashed_password: Mapped[str]
    email: Mapped[EmailStr] = mapped_column(String(100), unique=True, index=True)
    is_active: Mapped[bool] = mapped_column(Boolean, default=True)
    is_superuser: Mapped[bool] = mapped_column(Boolean, default=False)

    posts: Mapped[list["Post"]] = relationship(
        "Post",
        back_populates="author",
    )
    profile: Mapped["Profile"] = relationship(
        back_populates="user",
        lazy="selectin",
    )
    likes: Mapped[list["LikePost"]] = relationship(
        "LikePost",
        back_populates="user",
        cascade="all, delete-orphan",
        lazy="selectin",
    )
    comments: Mapped[list["Comment"] | None] = relationship(
        "Comment",
        back_populates="user",
        cascade="all, delete-orphan",
        lazy="selectin",
    )

    def __str__(self) -> str:
        return f"{self.__class__.__name__}(id={self.id}, username={self.username!r})"

    def __repr__(self) -> str:
        return str(self)
