from typing import cast

from backend.application.analytics_events import build_analytics_event_payload
from backend.application.event_types import COMMENT_LIKE_TOGGLED_EVENT
from backend.application.events import IntegrationEvent
from backend.application.ports.like_repository import LikeRepository
from backend.application.ports.outbox_repository import OutboxRepository
from backend.application.ports.transaction_manager import TransactionManager
from backend.application.results import ToggleLikeResult
from backend.domain.user.entity import User


class ToggleCommentLikeUseCase:
    def __init__(self, repository: LikeRepository, outbox: OutboxRepository, transaction_manager: TransactionManager) -> None:
        self.repository = repository
        self.outbox = outbox
        self.transaction_manager = transaction_manager

    async def execute(self, current_user: User, comment_id: int) -> ToggleLikeResult:
        user_id = cast(int, current_user.id)
        async with self.transaction_manager:
            result = await self.repository.toggle_comment_like(user_id, comment_id)
            await self.outbox.add(IntegrationEvent(
                event_type=COMMENT_LIKE_TOGGLED_EVENT,
                payload=build_analytics_event_payload(
                    user_id=user_id, entity_type="comment", entity_id=comment_id,
                    data={"action": result.action, "likes_count": result.likes_count},
                ),
            ))
        return ToggleLikeResult(result.success, result.action, result.likes_count, result.liked)
