"""
Модуль для хранения use case'ов, связанных с пользователями.
"""

from backend.application.ports.user_repository import UserRepository


class TouchUserActivityUseCase:
    """Record the latest activity of an authenticated user."""

    def __init__(self, user_repository: UserRepository) -> None:
        self.user_repository = user_repository

    async def execute(self, user_id: int) -> None:
        await self.user_repository.touch_last_seen(user_id)
