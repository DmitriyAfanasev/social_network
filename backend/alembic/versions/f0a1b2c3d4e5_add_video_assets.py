"""Add video assets and renditions.

Revision ID: f0a1b2c3d4e5
Revises: a1b2c3d4e5f6
"""

from collections.abc import Sequence
import sqlalchemy as sa
from alembic import op

revision: str = "f0a1b2c3d4e5"
down_revision: str | None = "a1b2c3d4e5f6"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    op.create_table(
        "video_albums",
        sa.Column("user_id", sa.Integer(), nullable=False),
        sa.Column("title", sa.String(length=80), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
        sa.Column("updated_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
        sa.Column("id", sa.Integer(), nullable=False),
        sa.ForeignKeyConstraint(["user_id"], ["users.id"], ondelete="CASCADE"),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_video_albums")),
    )
    op.create_index("ix_video_albums_user_id", "video_albums", ["user_id"], unique=False)
    op.create_table(
        "video_assets",
        sa.Column("media_id", sa.Integer(), nullable=False),
        sa.Column("album_id", sa.Integer(), nullable=True),
        sa.Column("status", sa.String(length=32), nullable=False, server_default="uploaded"),
        sa.Column("error_message", sa.String(length=2000), nullable=True),
        sa.Column("processed_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
        sa.Column("id", sa.Integer(), nullable=False),
        sa.ForeignKeyConstraint(["media_id"], ["media.id"], ondelete="CASCADE"),
        sa.ForeignKeyConstraint(["album_id"], ["video_albums.id"], ondelete="SET NULL"),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_video_assets")),
        sa.UniqueConstraint("media_id", name=op.f("uq_video_assets_media_id")),
    )
    op.create_index("ix_video_assets_media_id", "video_assets", ["media_id"], unique=False)
    op.create_index("ix_video_assets_album_id", "video_assets", ["album_id"], unique=False)
    op.create_index("ix_video_assets_status", "video_assets", ["status"], unique=False)
    op.create_table(
        "video_renditions",
        sa.Column("video_id", sa.Integer(), nullable=False),
        sa.Column("height", sa.Integer(), nullable=False),
        sa.Column("object_key", sa.String(length=512), nullable=False),
        sa.Column("content_type", sa.String(length=255), nullable=False, server_default="video/mp4"),
        sa.Column("size", sa.BigInteger(), nullable=False, server_default="0"),
        sa.Column("width", sa.Integer(), nullable=True),
        sa.Column("duration", sa.Float(), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
        sa.Column("id", sa.Integer(), nullable=False),
        sa.ForeignKeyConstraint(["video_id"], ["video_assets.id"], ondelete="CASCADE"),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_video_renditions")),
        sa.UniqueConstraint("object_key", name=op.f("uq_video_renditions_object_key")),
        sa.UniqueConstraint("video_id", "height", name=op.f("uq_video_renditions_video_height")),
    )
    op.create_index("ix_video_renditions_video_id", "video_renditions", ["video_id"], unique=False)


def downgrade() -> None:
    op.drop_index("ix_video_renditions_video_id", table_name="video_renditions")
    op.drop_table("video_renditions")
    op.drop_index("ix_video_assets_status", table_name="video_assets")
    op.drop_index("ix_video_assets_album_id", table_name="video_assets")
    op.drop_index("ix_video_assets_media_id", table_name="video_assets")
    op.drop_table("video_assets")
    op.drop_index("ix_video_albums_user_id", table_name="video_albums")
    op.drop_table("video_albums")
