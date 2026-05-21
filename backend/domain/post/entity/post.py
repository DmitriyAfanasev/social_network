from dataclasses import dataclass, field
from datetime import UTC, datetime
from typing import Protocol

from backend.domain.shared.base_entity import BaseEntity
from backend.domain.user.entity import User


class PostReadModelSource(Protocol):
    id: int
    content: str | None
    image: str | None
    author_id: int
    created_at: datetime
    updated_at: datetime


@dataclass(frozen=True, kw_only=True)
class PostDraft:
    content: str | None
    has_attachment: bool = False

    @classmethod
    def from_input(
        cls,
        content: str | None,
        *,
        has_attachment: bool,
    ) -> "PostDraft":
        normalized_content = content.strip() if content is not None else ""
        if normalized_content:
            return cls(content=normalized_content, has_attachment=has_attachment)
        if has_attachment:
            return cls(content=None, has_attachment=True)
        raise ValueError("Post must contain text or an attachment.")


@dataclass(kw_only=True)
class Post(BaseEntity):
    author_id: int
    content: str | None = None
    image: str | None = None
    is_published: bool = True
    updated_at: datetime = field(default_factory=lambda: datetime.now(UTC))

    @classmethod
    def from_read_model(cls, post: PostReadModelSource) -> "Post":
        return cls(
            id=post.id,
            content=post.content,
            image=post.image,
            author_id=post.author_id,
            created_at=post.created_at,
            updated_at=post.updated_at,
        )

    def can_be_managed_by(self, user: User) -> bool:
        return self.author_id == user.id or user.is_superuser

    def can_remove_image(self) -> bool:
        return bool(self.content and self.content.strip())
