"""Add profile music tracks.

Revision ID: 4d5e6f7a8b9c
Revises: 3c4d5e6f7a8b
"""

from collections.abc import Sequence

import sqlalchemy as sa
from alembic import op


revision: str = "4d5e6f7a8b9c"
down_revision: str | None = "3c4d5e6f7a8b"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    op.create_table(
        "music_tracks",
        sa.Column("id", sa.Integer(), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.func.now(), nullable=False),
        sa.Column("user_id", sa.Integer(), nullable=False),
        sa.Column("media_id", sa.Integer(), nullable=False),
        sa.Column("title", sa.String(length=200), nullable=False),
        sa.Column("artist", sa.String(length=120), server_default="", nullable=False),
        sa.ForeignKeyConstraint(["media_id"], ["media.id"], ondelete="CASCADE"),
        sa.ForeignKeyConstraint(["user_id"], ["users.id"], ondelete="CASCADE"),
        sa.PrimaryKeyConstraint("id"),
        sa.UniqueConstraint("media_id", name="uq_music_tracks_media_id"),
    )
    op.create_index("ix_music_tracks_user_id", "music_tracks", ["user_id"], unique=False)
    op.create_index("ix_music_tracks_user_created", "music_tracks", ["user_id", "created_at"], unique=False)


def downgrade() -> None:
    op.drop_index("ix_music_tracks_user_created", table_name="music_tracks")
    op.drop_index("ix_music_tracks_user_id", table_name="music_tracks")
    op.drop_table("music_tracks")
