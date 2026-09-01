"""Add user blocks.

Revision ID: 3b8d0a2c5e7f
Revises: 2a7c9e1f4b6d
"""
from collections.abc import Sequence

import sqlalchemy as sa
from alembic import op


revision: str = "3b8d0a2c5e7f"
down_revision: str | None = "2a7c9e1f4b6d"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    op.create_table(
        "user_blocks",
        sa.Column("blocker_id", sa.Integer(), nullable=False),
        sa.Column("blocked_id", sa.Integer(), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
        sa.Column("id", sa.Integer(), nullable=False),
        sa.ForeignKeyConstraint(["blocker_id"], ["users.id"], ondelete="CASCADE"),
        sa.ForeignKeyConstraint(["blocked_id"], ["users.id"], ondelete="CASCADE"),
        sa.CheckConstraint("blocker_id != blocked_id", name="check_user_blocks_not_self"),
        sa.PrimaryKeyConstraint("id"),
        sa.UniqueConstraint("blocker_id", "blocked_id", name="uq_user_blocks_pair"),
    )
    op.create_index("idx_user_blocks_blocker_id", "user_blocks", ["blocker_id"])
    op.create_index("idx_user_blocks_blocked_id", "user_blocks", ["blocked_id"])


def downgrade() -> None:
    op.drop_index("idx_user_blocks_blocked_id", table_name="user_blocks")
    op.drop_index("idx_user_blocks_blocker_id", table_name="user_blocks")
    op.drop_table("user_blocks")
