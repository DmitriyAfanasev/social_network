"""Add subscriptions

Revision ID: 93df6c3f824d
Revises: 14f6e1d6f7d1
Create Date: 2026-05-20 17:45:00.000000

"""
from collections.abc import Sequence

from alembic import op
import sqlalchemy as sa


revision: str = "93df6c3f824d"
down_revision: str | None = "14f6e1d6f7d1"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    op.create_table(
        "subscriptions",
        sa.Column("subscriber_id", sa.Integer(), nullable=False),
        sa.Column("target_id", sa.Integer(), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
        sa.Column("updated_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
        sa.Column("id", sa.Integer(), nullable=False),
        sa.CheckConstraint("subscriber_id != target_id", name=op.f("check_subscription_not_self")),
        sa.ForeignKeyConstraint(
            ["subscriber_id"],
            ["users.id"],
            name=op.f("fk_subscriptions_subscriber_id_users"),
            ondelete="CASCADE",
        ),
        sa.ForeignKeyConstraint(
            ["target_id"],
            ["users.id"],
            name=op.f("fk_subscriptions_target_id_users"),
            ondelete="CASCADE",
        ),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_subscriptions")),
        sa.UniqueConstraint("subscriber_id", "target_id", name="uq_subscriptions_subscriber_target"),
    )
    op.create_index("idx_subscriptions_subscriber_id", "subscriptions", ["subscriber_id"], unique=False)
    op.create_index("idx_subscriptions_target_id", "subscriptions", ["target_id"], unique=False)


def downgrade() -> None:
    op.drop_index("idx_subscriptions_target_id", table_name="subscriptions")
    op.drop_index("idx_subscriptions_subscriber_id", table_name="subscriptions")
    op.drop_table("subscriptions")
