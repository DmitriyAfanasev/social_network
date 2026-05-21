"""Add profile photo albums

Revision ID: c7a8d57f1e25
Revises: b6a8cf2d2b33
Create Date: 2026-05-20 18:45:00.000000

"""
from collections.abc import Sequence

from alembic import op
import sqlalchemy as sa


revision: str = "c7a8d57f1e25"
down_revision: str | None = "b6a8cf2d2b33"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    op.create_table(
        "profile_photo_albums",
        sa.Column("user_id", sa.Integer(), nullable=False),
        sa.Column("title", sa.String(length=80), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
        sa.Column("updated_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
        sa.Column("id", sa.Integer(), nullable=False),
        sa.ForeignKeyConstraint(
            ["user_id"],
            ["users.id"],
            name=op.f("fk_profile_photo_albums_user_id_users"),
            ondelete="CASCADE",
        ),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_profile_photo_albums")),
    )
    op.create_index("idx_profile_photo_albums_user_id", "profile_photo_albums", ["user_id"], unique=False)
    op.create_table(
        "profile_photos",
        sa.Column("album_id", sa.Integer(), nullable=False),
        sa.Column("user_id", sa.Integer(), nullable=False),
        sa.Column("photo_url", sa.Text(), nullable=False),
        sa.Column("caption", sa.Text(), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
        sa.Column("id", sa.Integer(), nullable=False),
        sa.ForeignKeyConstraint(
            ["album_id"],
            ["profile_photo_albums.id"],
            name=op.f("fk_profile_photos_album_id_profile_photo_albums"),
            ondelete="CASCADE",
        ),
        sa.ForeignKeyConstraint(
            ["user_id"],
            ["users.id"],
            name=op.f("fk_profile_photos_user_id_users"),
            ondelete="CASCADE",
        ),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_profile_photos")),
    )
    op.create_index("idx_profile_photos_album_id", "profile_photos", ["album_id"], unique=False)
    op.create_index("idx_profile_photos_user_id", "profile_photos", ["user_id"], unique=False)


def downgrade() -> None:
    op.drop_index("idx_profile_photos_user_id", table_name="profile_photos")
    op.drop_index("idx_profile_photos_album_id", table_name="profile_photos")
    op.drop_table("profile_photos")
    op.drop_index("idx_profile_photo_albums_user_id", table_name="profile_photo_albums")
    op.drop_table("profile_photo_albums")
