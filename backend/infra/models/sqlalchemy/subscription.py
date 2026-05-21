from typing import TYPE_CHECKING

from sqlalchemy import CheckConstraint, ForeignKey, Index, UniqueConstraint
from sqlalchemy.orm import Mapped, mapped_column, relationship

from .base import Base
from .mixins import TimestampsMixin


if TYPE_CHECKING:
    from .user import User


class Subscription(TimestampsMixin, Base):
    __tablename__ = "subscriptions"
    __table_args__ = (
        UniqueConstraint("subscriber_id", "target_id", name="uq_subscriptions_subscriber_target"),
        CheckConstraint("subscriber_id != target_id", name="check_subscription_not_self"),
        Index("idx_subscriptions_subscriber_id", "subscriber_id"),
        Index("idx_subscriptions_target_id", "target_id"),
    )

    subscriber_id: Mapped[int] = mapped_column(ForeignKey("users.id", ondelete="CASCADE"))
    target_id: Mapped[int] = mapped_column(ForeignKey("users.id", ondelete="CASCADE"))

    subscriber: Mapped["User"] = relationship("User", foreign_keys=[subscriber_id])
    target: Mapped["User"] = relationship("User", foreign_keys=[target_id])
