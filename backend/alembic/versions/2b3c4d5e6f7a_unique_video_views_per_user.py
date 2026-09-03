"""Make video views unique per authenticated user.

Revision ID: 2b3c4d5e6f7a
Revises: 1a2b3c4d5e6f
"""

from collections.abc import Sequence

import sqlalchemy as sa
from alembic import op


revision: str = "2b3c4d5e6f7a"
down_revision: str | None = "1a2b3c4d5e6f"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    # Existing installations may contain duplicate view events from the old
    # endpoint. Keep the first event for each video/user pair before adding
    # the database guarantee.
    op.execute(
        sa.text(
            "DELETE FROM video_views older "
            "USING video_views newer "
            "WHERE older.video_id = newer.video_id "
            "AND older.user_id = newer.user_id "
            "AND older.id > newer.id"
        )
    )
    op.create_index(
        "uq_video_views_video_user",
        "video_views",
        ["video_id", "user_id"],
        unique=True,
        postgresql_where=sa.text("user_id IS NOT NULL"),
    )


def downgrade() -> None:
    op.drop_index("uq_video_views_video_user", table_name="video_views")
