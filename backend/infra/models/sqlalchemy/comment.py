from typing import TYPE_CHECKING

from sqlalchemy import ForeignKey, Index, Integer, Text
from sqlalchemy.orm import Mapped, mapped_column, relationship

from .base import Base
from .mixins import TimestampsMixin


if TYPE_CHECKING:
    from .post import Post
    from .user import User


class Comment(TimestampsMixin, Base):
    """Модель комментария в посте."""

    __table_args__ = (
        Index("idx_comment_user_id", "user_id"),
        Index("idx_comment_post_id", "post_id"),
        Index("idx_comment_parent_id", "parent_id"),
    )

    user_id: Mapped[int] = mapped_column(
        Integer,
        ForeignKey("users.id", ondelete="CASCADE"),
        nullable=False,
    )
    post_id: Mapped[int] = mapped_column(
        Integer,
        ForeignKey("posts.id", ondelete="CASCADE"),
        nullable=False,
    )
    parent_id: Mapped[int | None] = mapped_column(
        Integer,
        ForeignKey("comments.id", ondelete="CASCADE"),
        nullable=True,
    )
    text: Mapped[str] = mapped_column(Text, nullable=False)

    # Связи
    user: Mapped["User"] = relationship("User", back_populates="comments")
    post: Mapped["Post"] = relationship("Post", back_populates="comments")

    parent: Mapped["Comment | None"] = relationship(
        "Comment", remote_side="Comment.id", back_populates="replies"
    )
    replies: Mapped[list["Comment"]] = relationship(
        "Comment", back_populates="parent", cascade="all, delete-orphan"
    )
