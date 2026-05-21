from datetime import datetime

from backend.application.events import JsonPayload
from sqlalchemy import DateTime, Index, Integer, String, Text
from sqlalchemy.dialects.postgresql import JSONB
from sqlalchemy.orm import Mapped, mapped_column

from .base import Base
from .mixins import TimestampsMixin


class OutboxEvent(TimestampsMixin, Base):
    __tablename__ = "outbox_events"
    __table_args__ = (
        Index("idx_outbox_events_status_next_retry_at", "status", "next_retry_at"),
        Index("idx_outbox_events_event_type", "event_type"),
    )

    event_type: Mapped[str] = mapped_column(String(120), nullable=False)
    payload: Mapped[JsonPayload] = mapped_column(JSONB, nullable=False)
    status: Mapped[str] = mapped_column(String(20), nullable=False, default="pending", server_default="pending")
    attempts: Mapped[int] = mapped_column(Integer, nullable=False, default=0, server_default="0")
    next_retry_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), nullable=False)
    published_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True))
    last_error: Mapped[str | None] = mapped_column(Text())
