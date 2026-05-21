from typing import TYPE_CHECKING

from sqlalchemy import Boolean, ForeignKey, Text
from sqlalchemy.orm import Mapped, mapped_column, relationship

from .base import Base
from .mixins import TimestampsMixin


if TYPE_CHECKING:
    from .comment import Comment
    from .like import LikePost
    from .user import User


class Post(TimestampsMixin, Base):
    """
    Модель для поста.
    """

    content: Mapped[str | None] = mapped_column(Text, nullable=True)
    is_published: Mapped[bool] = mapped_column(Boolean, default=True)
    author_id: Mapped[int] = mapped_column(ForeignKey("users.id"))

    author: Mapped["User"] = relationship(
        "User", back_populates="posts", lazy="selectin"
    )
    image: Mapped[str | None] = mapped_column(Text, server_default=None)

    likes: Mapped[list["LikePost"]] = relationship(
        back_populates="post", lazy="selectin"
    )

    comments: Mapped[list["Comment"] | None] = relationship(
        "Comment", back_populates="post"
    )
