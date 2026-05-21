from typing import TYPE_CHECKING

from sqlalchemy import CheckConstraint, ForeignKey, Index, UniqueConstraint
from sqlalchemy.orm import Mapped, mapped_column, relationship

from .base import Base
from .mixins import TimestampsMixin


if TYPE_CHECKING:
    from .user import User


class Friendship(TimestampsMixin, Base):
    __tablename__ = "friendships"
    __table_args__ = (
        UniqueConstraint("user_id", "friend_id", name="uq_friendships_user_friend"),
        CheckConstraint("user_id < friend_id", name="check_friendship_normalized_pair"),
        Index("idx_friendships_user_id", "user_id"),
        Index("idx_friendships_friend_id", "friend_id"),
    )

    user_id: Mapped[int] = mapped_column(ForeignKey("users.id", ondelete="CASCADE"))
    friend_id: Mapped[int] = mapped_column(ForeignKey("users.id", ondelete="CASCADE"))

    user: Mapped["User"] = relationship("User", foreign_keys=[user_id])
    friend: Mapped["User"] = relationship("User", foreign_keys=[friend_id])
