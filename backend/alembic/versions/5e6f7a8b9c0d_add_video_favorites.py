"""Add favorite video playlist.

Revision ID: 5e6f7a8b9c0d
Revises: 4d5e6f7a8b9c
"""

from collections.abc import Sequence

import sqlalchemy as sa
from alembic import op


revision: str = "5e6f7a8b9c0d"
down_revision: str | None = "4d5e6f7a8b9c"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    op.create_table(
        "video_favorites",
        sa.Column("video_id", sa.Integer(), nullable=False),
        sa.Column("user_id", sa.Integer(), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
        sa.Column("id", sa.Integer(), nullable=False),
        sa.ForeignKeyConstraint(["video_id"], ["video_assets.id"], ondelete="CASCADE"),
        sa.ForeignKeyConstraint(["user_id"], ["users.id"], ondelete="CASCADE"),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_video_favorites")),
        sa.UniqueConstraint("video_id", "user_id", name=op.f("uq_video_favorites_video_user")),
    )
    op.create_index("ix_video_favorites_video_id", "video_favorites", ["video_id"], unique=False)
    op.create_index("ix_video_favorites_user_id", "video_favorites", ["user_id"], unique=False)


def downgrade() -> None:
    op.drop_index("ix_video_favorites_user_id", table_name="video_favorites")
    op.drop_index("ix_video_favorites_video_id", table_name="video_favorites")
    op.drop_table("video_favorites")
