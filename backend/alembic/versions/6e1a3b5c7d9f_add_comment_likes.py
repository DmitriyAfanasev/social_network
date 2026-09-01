"""Add comment likes.

Revision ID: 6e1a3b5c7d9f
Revises: 5d0f2a3b7c9e
"""
from collections.abc import Sequence

import sqlalchemy as sa
from alembic import op


revision: str = "6e1a3b5c7d9f"
down_revision: str | None = "5d0f2a3b7c9e"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    op.create_table(
        "comment_likes",
        sa.Column("user_id", sa.Integer(), nullable=False),
        sa.Column("comment_id", sa.Integer(), nullable=False),
        sa.Column("id", sa.Integer(), nullable=False),
        sa.ForeignKeyConstraint(["comment_id"], ["comments.id"], ondelete="CASCADE"),
        sa.ForeignKeyConstraint(["user_id"], ["users.id"], ondelete="CASCADE"),
        sa.PrimaryKeyConstraint("id"),
        sa.UniqueConstraint("user_id", "comment_id", name="unique_user_comment"),
    )


def downgrade() -> None:
    op.drop_table("comment_likes")
