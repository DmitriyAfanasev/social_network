from typing import cast

from backend.application.analytics_events import build_analytics_event_payload
from backend.application.event_types import POST_LIKE_TOGGLED_EVENT
from backend.application.events import IntegrationEvent
from backend.application.ports.like_repository import LikeRepository
from backend.application.ports.outbox_repository import OutboxRepository
from backend.application.ports.transaction_manager import TransactionManager
from backend.application.results import ToggleLikeResult
from backend.domain.like import LikePost
from backend.domain.user.entity import User


class TogglePostLikeUseCase:
    def __init__(
        self,
        like_repository: LikeRepository,
        outbox_repository: OutboxRepository,
        transaction_manager: TransactionManager,
    ) -> None:
        self.like_repository = like_repository
        self.outbox_repository = outbox_repository
        self.transaction_manager = transaction_manager

    async def execute(self, current_user: User, post_id: int) -> ToggleLikeResult:
        current_user_id = cast(int, current_user.id)
        like = LikePost(user_id=current_user_id, post_id=post_id)
        async with self.transaction_manager:
            like_data = await self.like_repository.toggle_post_like(
                user_id=like.user_id,
                post_id=like.post_id,
            )
            await self.outbox_repository.add(
                IntegrationEvent(
                    event_type=POST_LIKE_TOGGLED_EVENT,
                    payload=build_analytics_event_payload(
                        user_id=current_user_id,
                        entity_type="post",
                        entity_id=post_id,
                        data={
                            "action": like_data.action,
                            "likes_count": like_data.likes_count,
                        },
                    ),
                )
            )
        return ToggleLikeResult(
            success=like_data.success,
            action=like_data.action,
            likes_count=like_data.likes_count,
            liked=like_data.liked,
        )
