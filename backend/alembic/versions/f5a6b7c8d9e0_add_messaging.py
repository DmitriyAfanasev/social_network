"""Add direct conversations and messages.

Revision ID: f5a6b7c8d9e0
Revises: e4b7c9d1a2f3
"""

from collections.abc import Sequence

import sqlalchemy as sa
from alembic import op


revision: str = "f5a6b7c8d9e0"
down_revision: str | None = "e4b7c9d1a2f3"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    op.create_table(
        "conversations",
        sa.Column("direct_key", sa.String(length=64), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
        sa.Column("id", sa.Integer(), nullable=False),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_conversations")),
        sa.UniqueConstraint("direct_key", name=op.f("uq_conversations_direct_key")),
    )
    op.create_index("ix_conversations_direct_key", "conversations", ["direct_key"], unique=False)
    op.create_table(
        "messages",
        sa.Column("conversation_id", sa.Integer(), nullable=False),
        sa.Column("sender_id", sa.Integer(), nullable=False),
        sa.Column("text", sa.Text(), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
        sa.Column("id", sa.Integer(), nullable=False),
        sa.ForeignKeyConstraint(["conversation_id"], ["conversations.id"], ondelete="CASCADE"),
        sa.ForeignKeyConstraint(["sender_id"], ["users.id"], ondelete="CASCADE"),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_messages")),
    )
    op.create_index("idx_messages_conversation_created", "messages", ["conversation_id", "created_at", "id"])
    op.create_table(
        "conversation_participants",
        sa.Column("conversation_id", sa.Integer(), nullable=False),
        sa.Column("user_id", sa.Integer(), nullable=False),
        sa.Column("last_read_message_id", sa.Integer(), nullable=True),
        sa.Column("read_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
        sa.Column("id", sa.Integer(), nullable=False),
        sa.ForeignKeyConstraint(["conversation_id"], ["conversations.id"], ondelete="CASCADE"),
        sa.ForeignKeyConstraint(["user_id"], ["users.id"], ondelete="CASCADE"),
        sa.ForeignKeyConstraint(["last_read_message_id"], ["messages.id"], ondelete="SET NULL"),
        sa.PrimaryKeyConstraint("id", name=op.f("pk_conversation_participants")),
        sa.UniqueConstraint("conversation_id", "user_id", name="uq_conversation_participant"),
    )
    op.create_index("idx_conversation_participants_user_id", "conversation_participants", ["user_id"])


def downgrade() -> None:
    op.drop_index("idx_conversation_participants_user_id", table_name="conversation_participants")
    op.drop_table("conversation_participants")
    op.drop_index("idx_messages_conversation_created", table_name="messages")
    op.drop_table("messages")
    op.drop_index("ix_conversations_direct_key", table_name="conversations")
    op.drop_table("conversations")
