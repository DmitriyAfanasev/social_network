from pydantic import BaseModel, Field


class CommentCreateRequest(BaseModel):
    content: str = Field(min_length=1)
    parent_id: int | None = None


class CommentUpdateRequest(BaseModel):
    content: str = Field(min_length=1)
