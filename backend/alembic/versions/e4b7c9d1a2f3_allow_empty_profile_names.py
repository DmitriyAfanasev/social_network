"""Allow profiles to be created before personal data is filled in.

Revision ID: e4b7c9d1a2f3
Revises: d3f2a8b4c901
"""

from collections.abc import Sequence

from alembic import op


revision: str = "e4b7c9d1a2f3"
down_revision: str | None = "d3f2a8b4c901"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    op.alter_column("profiles", "first_name", nullable=True)
    op.alter_column("profiles", "last_name", nullable=True)


def downgrade() -> None:
    op.alter_column("profiles", "first_name", nullable=False)
    op.alter_column("profiles", "last_name", nullable=False)
