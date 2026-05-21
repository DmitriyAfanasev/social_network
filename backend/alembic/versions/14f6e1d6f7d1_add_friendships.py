"""Add friendships

Revision ID: 14f6e1d6f7d1
Revises: a775b020fd78
Create Date: 2026-05-20 14:45:00.000000

"""
from collections.abc import Sequence

from alembic import op
import sqlalchemy as sa


revision: str = "14f6e1d6f7d1"
down_revision: str | None = "a775b020fd78"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    op.create_table(
        "friendships",
        sa.Column("user_id", sa.Integer(), nullable=False),
        sa.Column("friend_id", sa.Integer(), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
        sa.Column("updated_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
        sa.Column("id", sa.Integer(), nullable=False),
        sa.CheckConstraint("user_id < friend_id", name=op.f("check_friendship_normalized_pair")),
        sa.ForeignKeyConstraint(["friend_id"], ["users.id"], name=op.f("fk_friendships_friend_id_users"), ondelete="CASCADE"),
        sa.ForeignKeyConstraint(["user_id"], ["users.id"], name=op.f("fk_friendships_user_id_users"), ondelete="CASCADE"),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_friendships")),
        sa.UniqueConstraint("user_id", "friend_id", name="uq_friendships_user_friend"),
    )
    op.create_index("idx_friendships_friend_id", "friendships", ["friend_id"], unique=False)
    op.create_index("idx_friendships_user_id", "friendships", ["user_id"], unique=False)


def downgrade() -> None:
    op.drop_index("idx_friendships_user_id", table_name="friendships")
    op.drop_index("idx_friendships_friend_id", table_name="friendships")
    op.drop_table("friendships")
