"""Add profile avatar history

Revision ID: b6a8cf2d2b33
Revises: 93df6c3f824d
Create Date: 2026-05-20 18:20:00.000000

"""
from collections.abc import Sequence

from alembic import op
import sqlalchemy as sa


revision: str = "b6a8cf2d2b33"
down_revision: str | None = "93df6c3f824d"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    op.create_table(
        "profile_avatars",
        sa.Column("user_id", sa.Integer(), nullable=False),
        sa.Column("avatar_url", sa.Text(), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
        sa.Column("id", sa.Integer(), nullable=False),
        sa.ForeignKeyConstraint(
            ["user_id"],
            ["users.id"],
            name=op.f("fk_profile_avatars_user_id_users"),
            ondelete="CASCADE",
        ),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_profile_avatars")),
        sa.UniqueConstraint("user_id", "avatar_url", name="uq_profile_avatars_user_avatar"),
    )
    op.create_index("idx_profile_avatars_user_id", "profile_avatars", ["user_id"], unique=False)


def downgrade() -> None:
    op.drop_index("idx_profile_avatars_user_id", table_name="profile_avatars")
    op.drop_table("profile_avatars")
