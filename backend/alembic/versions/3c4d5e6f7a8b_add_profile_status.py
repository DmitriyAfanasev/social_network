"""Add a short profile status.

Revision ID: 3c4d5e6f7a8b
Revises: 2b3c4d5e6f7a
"""

from collections.abc import Sequence

import sqlalchemy as sa
from alembic import op


revision: str = "3c4d5e6f7a8b"
down_revision: str | None = "2b3c4d5e6f7a"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    op.add_column("profiles", sa.Column("status", sa.String(length=140), nullable=True))


def downgrade() -> None:
    op.drop_column("profiles", "status")
