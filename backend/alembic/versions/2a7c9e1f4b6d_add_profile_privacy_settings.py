"""Add profile privacy settings.

Revision ID: 2a7c9e1f4b6d
Revises: a1b2c3d4e5f6
"""
from collections.abc import Sequence

import sqlalchemy as sa
from alembic import op


revision: str = "2a7c9e1f4b6d"
down_revision: str | None = "a1b2c3d4e5f6"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    op.add_column("profiles", sa.Column("profile_visibility", sa.String(length=24), server_default="everyone", nullable=False))
    op.add_column("profiles", sa.Column("friend_request_policy", sa.String(length=24), server_default="everyone", nullable=False))
    op.add_column("profiles", sa.Column("message_policy", sa.String(length=24), server_default="everyone", nullable=False))
    op.add_column("profiles", sa.Column("show_email", sa.Boolean(), server_default=sa.false(), nullable=False))
    op.add_column("profiles", sa.Column("show_phone", sa.Boolean(), server_default=sa.false(), nullable=False))
    op.add_column("profiles", sa.Column("show_birth_date", sa.Boolean(), server_default=sa.true(), nullable=False))
    op.add_column("profiles", sa.Column("show_friends", sa.Boolean(), server_default=sa.true(), nullable=False))
    op.add_column("profiles", sa.Column("show_posts", sa.Boolean(), server_default=sa.true(), nullable=False))


def downgrade() -> None:
    for name in ("show_posts", "show_friends", "show_birth_date", "show_phone", "show_email", "message_policy", "friend_request_policy", "profile_visibility"):
        op.drop_column("profiles", name)
