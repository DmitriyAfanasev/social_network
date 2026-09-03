"""Merge the existing application and video migration branches.

Revision ID: 0c1d2e3f4a5b
Revises: 9b4d6e8f2a10, f0a1b2c3d4e5
"""

from collections.abc import Sequence


revision: str = "0c1d2e3f4a5b"
down_revision: tuple[str, str] = ("9b4d6e8f2a10", "f0a1b2c3d4e5")
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    """Merge migration heads; both branches already applied their schema changes."""


def downgrade() -> None:
    """Recreate the two historical heads when downgrading the merge."""
