from pydantic import BaseModel, Field


class DirectConversationRequest(BaseModel):
    user_id: int = Field(gt=0)


class SendMessageRequest(BaseModel):
    text: str = Field(min_length=1, max_length=5000)


class EditMessageRequest(BaseModel):
    """Данные для изменения текста сообщения."""

    text: str = Field(min_length=1, max_length=5000)


class MarkReadRequest(BaseModel):
    message_id: int = Field(gt=0)
