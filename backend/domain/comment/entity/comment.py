from dataclasses import dataclass, field
from datetime import UTC, datetime
from typing import Protocol

from backend.domain.shared.base_entity import BaseEntity


class CommentReadModelSource(Protocol):
    id: int
    user_id: int
    post_id: int
    parent_id: int | None
    text: str
    created_at: datetime
    updated_at: datetime


@dataclass(kw_only=True)
class Comment(BaseEntity):
    user_id: int
    post_id: int
    text: str
    parent_id: int | None = None
    updated_at: datetime = field(default_factory=lambda: datetime.now(UTC))

    @classmethod
    def from_read_model(cls, comment: CommentReadModelSource) -> "Comment":
        return cls(
            id=comment.id,
            user_id=comment.user_id,
            post_id=comment.post_id,
            parent_id=comment.parent_id,
            text=comment.text,
            created_at=comment.created_at,
            updated_at=comment.updated_at,
        )

    def can_accept_reply_for_post(self, post_id: int) -> bool:
        return self.parent_id is None and self.post_id == post_id
