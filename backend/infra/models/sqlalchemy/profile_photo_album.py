from typing import TYPE_CHECKING

from sqlalchemy import ForeignKey, Index, String
from sqlalchemy.orm import Mapped, mapped_column, relationship

from .base import Base
from .mixins import TimestampsMixin


if TYPE_CHECKING:
    from .profile_photo import ProfilePhoto
    from .user import User


class ProfilePhotoAlbum(TimestampsMixin, Base):
    __tablename__ = "profile_photo_albums"
    __table_args__ = (Index("idx_profile_photo_albums_user_id", "user_id"),)

    user_id: Mapped[int] = mapped_column(ForeignKey("users.id", ondelete="CASCADE"))
    title: Mapped[str] = mapped_column(String(80))

    user: Mapped["User"] = relationship("User")
    photos: Mapped[list["ProfilePhoto"]] = relationship(
        "ProfilePhoto",
        back_populates="album",
        cascade="all, delete-orphan",
        lazy="selectin",
    )
