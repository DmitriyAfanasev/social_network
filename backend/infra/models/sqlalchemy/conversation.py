import datetime
from typing import TYPE_CHECKING

from sqlalchemy import DateTime, ForeignKey, Index, String, Text, UniqueConstraint
from sqlalchemy.orm import Mapped, mapped_column, query_expression, relationship

from .base import Base
from .mixins import CreatedAtMixin


if TYPE_CHECKING:
    from .media import Media
    from .user import User


class Conversation(CreatedAtMixin, Base):
    __tablename__ = "conversations"

    last_message: Mapped["Message | None"] = query_expression()
    direct_key: Mapped[str] = mapped_column(String(64), unique=True, index=True)
    participants: Mapped[list["ConversationParticipant"]] = relationship(
        back_populates="conversation", cascade="all, delete-orphan"
    )
    messages: Mapped[list["Message"]] = relationship(
        back_populates="conversation", cascade="all, delete-orphan"
    )


class ConversationParticipant(CreatedAtMixin, Base):
    __tablename__ = "conversation_participants"
    __table_args__ = (
        UniqueConstraint("conversation_id", "user_id", name="uq_conversation_participant"),
        Index("idx_conversation_participants_user_id", "user_id"),
    )

    conversation_id: Mapped[int] = mapped_column(
        ForeignKey("conversations.id", ondelete="CASCADE")
    )
    user_id: Mapped[int] = mapped_column(ForeignKey("users.id", ondelete="CASCADE"))
    last_read_message_id: Mapped[int | None] = mapped_column(
        ForeignKey("messages.id", ondelete="SET NULL")
    )
    read_at: Mapped[datetime.datetime | None] = mapped_column(nullable=True)
    archived_at: Mapped[datetime.datetime | None] = mapped_column(nullable=True)
    hidden_at: Mapped[datetime.datetime | None] = mapped_column(nullable=True)
    pinned_at: Mapped[datetime.datetime | None] = mapped_column(nullable=True)
    muted_at: Mapped[datetime.datetime | None] = mapped_column(nullable=True)
    cleared_at: Mapped[datetime.datetime | None] = mapped_column(nullable=True)

    conversation: Mapped[Conversation] = relationship(back_populates="participants")
    user: Mapped["User"] = relationship()


class Message(CreatedAtMixin, Base):
    __tablename__ = "messages"
    __table_args__ = (
        Index("idx_messages_conversation_created", "conversation_id", "created_at", "id"),
    )

    conversation_id: Mapped[int] = mapped_column(
        ForeignKey("conversations.id", ondelete="CASCADE")
    )
    sender_id: Mapped[int] = mapped_column(ForeignKey("users.id", ondelete="CASCADE"))
    text: Mapped[str] = mapped_column(Text())
    media_id: Mapped[int | None] = mapped_column(ForeignKey("media.id", ondelete="SET NULL"), nullable=True)
    edited_at: Mapped[datetime.datetime | None] = mapped_column(DateTime(timezone=True), nullable=True)
    deleted_at: Mapped[datetime.datetime | None] = mapped_column(DateTime(timezone=True), nullable=True)
    media: Mapped["Media | None"] = relationship(lazy="selectin")

    conversation: Mapped[Conversation] = relationship(back_populates="messages")
    sender: Mapped["User"] = relationship()
