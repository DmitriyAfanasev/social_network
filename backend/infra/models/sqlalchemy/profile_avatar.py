from typing import TYPE_CHECKING

from sqlalchemy import ForeignKey, Index, Text, UniqueConstraint
from sqlalchemy.orm import Mapped, mapped_column, relationship

from .base import Base
from .mixins import CreatedAtMixin


if TYPE_CHECKING:
    from .user import User


class ProfileAvatar(CreatedAtMixin, Base):
    __tablename__ = "profile_avatars"
    __table_args__ = (
        UniqueConstraint("user_id", "avatar_url", name="uq_profile_avatars_user_avatar"),
        Index("idx_profile_avatars_user_id", "user_id"),
    )

    user_id: Mapped[int] = mapped_column(ForeignKey("users.id", ondelete="CASCADE"))
    avatar_url: Mapped[str] = mapped_column(Text())

    user: Mapped["User"] = relationship("User")
