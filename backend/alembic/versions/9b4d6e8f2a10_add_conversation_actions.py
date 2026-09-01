"""Add per-user conversation actions.

Revision ID: 9b4d6e8f2a10
Revises: 8a3c5e7f1b9d
"""

from alembic import op
import sqlalchemy as sa


revision = "9b4d6e8f2a10"
down_revision = "8a3c5e7f1b9d"
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.add_column("conversation_participants", sa.Column("pinned_at", sa.DateTime(), nullable=True))
    op.add_column("conversation_participants", sa.Column("muted_at", sa.DateTime(), nullable=True))
    op.add_column("conversation_participants", sa.Column("cleared_at", sa.DateTime(), nullable=True))


def downgrade() -> None:
    op.drop_column("conversation_participants", "cleared_at")
    op.drop_column("conversation_participants", "muted_at")
    op.drop_column("conversation_participants", "pinned_at")
