"""Add per-user archive and hide state to conversations."""

from collections.abc import Sequence

import sqlalchemy as sa
from alembic import op


revision: str = "7f2a4c8d1e6b"
down_revision: str | None = "6e1a3b5c7d9f"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    op.add_column("conversation_participants", sa.Column("archived_at", sa.DateTime(timezone=False), nullable=True))
    op.add_column("conversation_participants", sa.Column("hidden_at", sa.DateTime(timezone=False), nullable=True))


def downgrade() -> None:
    op.drop_column("conversation_participants", "hidden_at")
    op.drop_column("conversation_participants", "archived_at")
