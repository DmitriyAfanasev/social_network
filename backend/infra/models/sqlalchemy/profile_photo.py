from typing import TYPE_CHECKING

from sqlalchemy import ForeignKey, Index, Text
from sqlalchemy.orm import Mapped, mapped_column, relationship

from .base import Base
from .mixins import CreatedAtMixin


if TYPE_CHECKING:
    from .profile_photo_album import ProfilePhotoAlbum
    from .user import User


class ProfilePhoto(CreatedAtMixin, Base):
    __tablename__ = "profile_photos"
    __table_args__ = (
        Index("idx_profile_photos_album_id", "album_id"),
        Index("idx_profile_photos_user_id", "user_id"),
    )

    album_id: Mapped[int] = mapped_column(ForeignKey("profile_photo_albums.id", ondelete="CASCADE"))
    user_id: Mapped[int] = mapped_column(ForeignKey("users.id", ondelete="CASCADE"))
    photo_url: Mapped[str] = mapped_column(Text())
    caption: Mapped[str | None] = mapped_column(Text())

    album: Mapped["ProfilePhotoAlbum"] = relationship("ProfilePhotoAlbum", back_populates="photos")
    user: Mapped["User"] = relationship("User")
