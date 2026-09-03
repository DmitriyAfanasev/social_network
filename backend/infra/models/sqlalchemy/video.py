from datetime import datetime

from sqlalchemy import BigInteger, DateTime, ForeignKey, Index, Integer, String, UniqueConstraint, text
from sqlalchemy.orm import Mapped, mapped_column, relationship

from .base import Base
from .mixins import CreatedAtMixin


class VideoAsset(CreatedAtMixin, Base):
    __tablename__ = "video_assets"
    __table_args__ = (UniqueConstraint("media_id", name="uq_video_assets_media_id"),)

    media_id: Mapped[int] = mapped_column(ForeignKey("media.id", ondelete="CASCADE"), index=True)
    album_id: Mapped[int | None] = mapped_column(ForeignKey("video_albums.id", ondelete="SET NULL"), nullable=True, index=True)
    title: Mapped[str] = mapped_column(String(200), default="Видео", server_default="Видео")
    status: Mapped[str] = mapped_column(String(32), default="uploaded", index=True)
    error_message: Mapped[str | None] = mapped_column(String(2000), nullable=True)
    processed_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True), nullable=True)
    album: Mapped["VideoAlbum | None"] = relationship("VideoAlbum", back_populates="videos")


class VideoRendition(CreatedAtMixin, Base):
    __tablename__ = "video_renditions"
    __table_args__ = (UniqueConstraint("video_id", "height", name="uq_video_renditions_video_height"),)

    video_id: Mapped[int] = mapped_column(ForeignKey("video_assets.id", ondelete="CASCADE"), index=True)
    height: Mapped[int] = mapped_column(Integer())
    object_key: Mapped[str] = mapped_column(String(512), unique=True)
    content_type: Mapped[str] = mapped_column(String(255), default="video/mp4")
    size: Mapped[int] = mapped_column(BigInteger(), default=0)
    width: Mapped[int | None] = mapped_column(Integer(), nullable=True)
    duration: Mapped[float | None] = mapped_column(nullable=True)


class VideoLike(CreatedAtMixin, Base):
    __tablename__ = "video_likes"
    __table_args__ = (UniqueConstraint("video_id", "user_id", name="uq_video_likes_video_user"),)

    video_id: Mapped[int] = mapped_column(ForeignKey("video_assets.id", ondelete="CASCADE"), index=True)
    user_id: Mapped[int] = mapped_column(ForeignKey("users.id", ondelete="CASCADE"), index=True)


class VideoFavorite(CreatedAtMixin, Base):
    __tablename__ = "video_favorites"
    __table_args__ = (UniqueConstraint("video_id", "user_id", name="uq_video_favorites_video_user"),)

    video_id: Mapped[int] = mapped_column(ForeignKey("video_assets.id", ondelete="CASCADE"), index=True)
    user_id: Mapped[int] = mapped_column(ForeignKey("users.id", ondelete="CASCADE"), index=True)


class VideoViewRecord(CreatedAtMixin, Base):
    __tablename__ = "video_views"
    __table_args__ = (
        Index(
            "uq_video_views_video_user",
            "video_id",
            "user_id",
            unique=True,
            postgresql_where=text("user_id IS NOT NULL"),
        ),
    )

    video_id: Mapped[int] = mapped_column(ForeignKey("video_assets.id", ondelete="CASCADE"), index=True)
    user_id: Mapped[int | None] = mapped_column(ForeignKey("users.id", ondelete="SET NULL"), nullable=True, index=True)


class VideoBookmark(CreatedAtMixin, Base):
    __tablename__ = "video_bookmarks"
    __table_args__ = (UniqueConstraint("video_id", "user_id", name="uq_video_bookmarks_video_user"),)

    video_id: Mapped[int] = mapped_column(ForeignKey("video_assets.id", ondelete="CASCADE"), index=True)
    user_id: Mapped[int] = mapped_column(ForeignKey("users.id", ondelete="CASCADE"), index=True)


class VideoAlbum(CreatedAtMixin, Base):
    __tablename__ = "video_albums"

    user_id: Mapped[int] = mapped_column(ForeignKey("users.id", ondelete="CASCADE"), index=True)
    title: Mapped[str] = mapped_column(String(80))
    videos: Mapped[list[VideoAsset]] = relationship("VideoAsset", back_populates="album", lazy="selectin")
