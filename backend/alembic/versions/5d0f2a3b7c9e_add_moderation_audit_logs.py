"""Add moderation audit logs.

Revision ID: 5d0f2a3b7c9e
Revises: 4c9e1b2d6f8a
"""
from collections.abc import Sequence

import sqlalchemy as sa
from alembic import op


revision: str = "5d0f2a3b7c9e"
down_revision: str | None = "4c9e1b2d6f8a"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    op.create_table(
        "moderation_audit_logs",
        sa.Column("actor_id", sa.Integer(), nullable=True),
        sa.Column("action", sa.String(length=100), nullable=False),
        sa.Column("target_type", sa.String(length=50), nullable=False),
        sa.Column("target_id", sa.Integer(), nullable=True),
        sa.Column("details", sa.JSON(), server_default=sa.text("'{}'"), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
        sa.Column("id", sa.Integer(), nullable=False),
        sa.ForeignKeyConstraint(["actor_id"], ["users.id"], ondelete="SET NULL"),
        sa.PrimaryKeyConstraint("id"),
    )
    op.create_index("idx_moderation_audit_created_at", "moderation_audit_logs", ["created_at"])
    op.create_index("idx_moderation_audit_actor_id", "moderation_audit_logs", ["actor_id"])


def downgrade() -> None:
    op.drop_index("idx_moderation_audit_actor_id", table_name="moderation_audit_logs")
    op.drop_index("idx_moderation_audit_created_at", table_name="moderation_audit_logs")
    op.drop_table("moderation_audit_logs")
