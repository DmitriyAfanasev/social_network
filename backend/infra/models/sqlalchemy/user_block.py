from sqlalchemy import CheckConstraint, ForeignKey, Index, UniqueConstraint
from sqlalchemy.orm import Mapped, mapped_column

from .base import Base
from .mixins import CreatedAtMixin


class UserBlock(CreatedAtMixin, Base):
    """A directed block: the blocker no longer interacts with the blocked user."""

    __tablename__ = "user_blocks"
    __table_args__ = (
        UniqueConstraint("blocker_id", "blocked_id", name="uq_user_blocks_pair"),
        CheckConstraint("blocker_id != blocked_id", name="check_user_blocks_not_self"),
        Index("idx_user_blocks_blocker_id", "blocker_id"),
        Index("idx_user_blocks_blocked_id", "blocked_id"),
    )

    blocker_id: Mapped[int] = mapped_column(ForeignKey("users.id", ondelete="CASCADE"))
    blocked_id: Mapped[int] = mapped_column(ForeignKey("users.id", ondelete="CASCADE"))
