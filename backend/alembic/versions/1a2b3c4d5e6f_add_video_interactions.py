"""Add video titles, likes, views and bookmarks.

Revision ID: 1a2b3c4d5e6f
Revises: 0c1d2e3f4a5b
"""

from collections.abc import Sequence

import sqlalchemy as sa
from alembic import op


revision: str = "1a2b3c4d5e6f"
down_revision: str | None = "0c1d2e3f4a5b"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    op.add_column(
        "video_assets",
        sa.Column("title", sa.String(length=200), nullable=False, server_default="Видео"),
    )
    op.execute(
        "UPDATE video_assets SET title = COALESCE(NULLIF(media.original_filename, ''), 'Видео') "
        "FROM media WHERE media.id = video_assets.media_id"
    )
    op.alter_column("video_assets", "title", server_default=None)

    op.create_table(
        "video_likes",
        sa.Column("video_id", sa.Integer(), nullable=False),
        sa.Column("user_id", sa.Integer(), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
        sa.Column("id", sa.Integer(), nullable=False),
        sa.ForeignKeyConstraint(["video_id"], ["video_assets.id"], ondelete="CASCADE"),
        sa.ForeignKeyConstraint(["user_id"], ["users.id"], ondelete="CASCADE"),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_video_likes")),
        sa.UniqueConstraint("video_id", "user_id", name=op.f("uq_video_likes_video_user")),
    )
    op.create_index("ix_video_likes_video_id", "video_likes", ["video_id"], unique=False)
    op.create_index("ix_video_likes_user_id", "video_likes", ["user_id"], unique=False)

    op.create_table(
        "video_views",
        sa.Column("video_id", sa.Integer(), nullable=False),
        sa.Column("user_id", sa.Integer(), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
        sa.Column("id", sa.Integer(), nullable=False),
        sa.ForeignKeyConstraint(["video_id"], ["video_assets.id"], ondelete="CASCADE"),
        sa.ForeignKeyConstraint(["user_id"], ["users.id"], ondelete="SET NULL"),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_video_views")),
    )
    op.create_index("ix_video_views_video_id", "video_views", ["video_id"], unique=False)
    op.create_index("ix_video_views_user_id", "video_views", ["user_id"], unique=False)

    op.create_table(
        "video_bookmarks",
        sa.Column("video_id", sa.Integer(), nullable=False),
        sa.Column("user_id", sa.Integer(), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
        sa.Column("id", sa.Integer(), nullable=False),
        sa.ForeignKeyConstraint(["video_id"], ["video_assets.id"], ondelete="CASCADE"),
        sa.ForeignKeyConstraint(["user_id"], ["users.id"], ondelete="CASCADE"),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_video_bookmarks")),
        sa.UniqueConstraint("video_id", "user_id", name=op.f("uq_video_bookmarks_video_user")),
    )
    op.create_index("ix_video_bookmarks_video_id", "video_bookmarks", ["video_id"], unique=False)
    op.create_index("ix_video_bookmarks_user_id", "video_bookmarks", ["user_id"], unique=False)


def downgrade() -> None:
    op.drop_index("ix_video_bookmarks_user_id", table_name="video_bookmarks")
    op.drop_index("ix_video_bookmarks_video_id", table_name="video_bookmarks")
    op.drop_table("video_bookmarks")
    op.drop_index("ix_video_views_user_id", table_name="video_views")
    op.drop_index("ix_video_views_video_id", table_name="video_views")
    op.drop_table("video_views")
    op.drop_index("ix_video_likes_user_id", table_name="video_likes")
    op.drop_index("ix_video_likes_video_id", table_name="video_likes")
    op.drop_table("video_likes")
    op.drop_column("video_assets", "title")
