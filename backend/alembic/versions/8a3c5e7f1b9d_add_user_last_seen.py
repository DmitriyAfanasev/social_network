"""Add last activity timestamp to users."""

from collections.abc import Sequence

import sqlalchemy as sa
from alembic import op


revision: str = "8a3c5e7f1b9d"
down_revision: str | None = "7f2a4c8d1e6b"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    op.add_column(
        "users",
        sa.Column("last_seen_at", sa.DateTime(timezone=True), nullable=True),
    )


def downgrade() -> None:
    op.drop_column("users", "last_seen_at")
