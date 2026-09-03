from sqlalchemy import ForeignKey, Index, String, UniqueConstraint
from sqlalchemy.orm import Mapped, mapped_column

from .base import Base
from .mixins import CreatedAtMixin


class MusicTrack(CreatedAtMixin, Base):
    __tablename__ = "music_tracks"
    __table_args__ = (
        UniqueConstraint("media_id", name="uq_music_tracks_media_id"),
        Index("ix_music_tracks_user_created", "user_id", "created_at"),
    )

    user_id: Mapped[int] = mapped_column(ForeignKey("users.id", ondelete="CASCADE"), index=True)
    media_id: Mapped[int] = mapped_column(ForeignKey("media.id", ondelete="CASCADE"))
    title: Mapped[str] = mapped_column(String(200))
    artist: Mapped[str] = mapped_column(String(120), default="", server_default="")
