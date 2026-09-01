"""Add media metadata and message actions.

Revision ID: a1b2c3d4e5f6
Revises: f5a6b7c8d9e0
"""

from collections.abc import Sequence

import sqlalchemy as sa
from alembic import op


revision: str = "a1b2c3d4e5f6"
down_revision: str | None = "f5a6b7c8d9e0"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    """Создаёт таблицу медиа и добавляет состояния сообщения."""
    op.create_table(
        "media",
        sa.Column("object_key", sa.String(length=512), nullable=False),
        sa.Column("bucket", sa.String(length=255), nullable=False),
        sa.Column("original_filename", sa.String(length=255), nullable=False),
        sa.Column("content_type", sa.String(length=255), nullable=True),
        sa.Column("media_type", sa.String(length=32), nullable=False),
        sa.Column("size", sa.BigInteger(), nullable=False),
        sa.Column("checksum", sa.String(length=64), nullable=True),
        sa.Column("width", sa.Integer(), nullable=True),
        sa.Column("height", sa.Integer(), nullable=True),
        sa.Column("duration", sa.Float(), nullable=True),
        sa.Column("uploaded_by", sa.Integer(), nullable=True),
        sa.Column("deleted_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
        sa.Column("id", sa.Integer(), nullable=False),
        sa.ForeignKeyConstraint(["uploaded_by"], ["users.id"], ondelete="SET NULL"),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_media")),
        sa.UniqueConstraint("object_key", name=op.f("uq_media_object_key")),
    )
    op.create_index("ix_media_object_key", "media", ["object_key"], unique=True)
    op.create_index("ix_media_checksum", "media", ["checksum"], unique=False)
    op.add_column("messages", sa.Column("media_id", sa.Integer(), nullable=True))
    op.add_column("messages", sa.Column("edited_at", sa.DateTime(timezone=True), nullable=True))
    op.add_column("messages", sa.Column("deleted_at", sa.DateTime(timezone=True), nullable=True))
    op.create_foreign_key(
        "fk_messages_media_id_media", "messages", "media", ["media_id"], ["id"], ondelete="SET NULL"
    )


def downgrade() -> None:
    """Удаляет поля действий сообщений и таблицу медиа."""
    op.drop_constraint("fk_messages_media_id_media", "messages", type_="foreignkey")
    op.drop_column("messages", "deleted_at")
    op.drop_column("messages", "edited_at")
    op.drop_column("messages", "media_id")
    op.drop_index("ix_media_checksum", table_name="media")
    op.drop_index("ix_media_object_key", table_name="media")
    op.drop_table("media")
