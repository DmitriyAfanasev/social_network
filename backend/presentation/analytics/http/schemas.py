from pydantic import BaseModel, Field


class AnalyticsSummaryResponse(BaseModel):
    events_count: int = Field(description="Общее количество analytics events в ClickHouse.")
    unique_users: int = Field(description="Количество уникальных user_id в analytics events.")
    users_registered: int = Field(description="Количество событий регистрации пользователей.")
    posts_created: int = Field(description="Количество событий создания постов.")
    comments_created: int = Field(description="Количество событий создания комментариев.")
    likes_added: int = Field(description="Количество добавленных лайков.")
    likes_removed: int = Field(description="Количество снятых лайков.")
    photos_deleted: int = Field(description="Количество удалённых фотографий профиля.")
