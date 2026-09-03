from backend.application.exceptions import NotFoundError, ValidationAppError
from backend.application.ports.audit_repository import AuditRepository
from backend.application.ports.block_repository import BlockRepository
from backend.application.ports.friend_repository import FriendRepository
from backend.application.ports.transaction_manager import TransactionManager
from backend.application.results import MessageResult
from backend.domain.user.entity import User


class BlockUserUseCase:
    def __init__(self, repository: BlockRepository, friend_repository: FriendRepository, transaction_manager: TransactionManager, audit_repository: AuditRepository | None = None) -> None:
        self.repository = repository
        self.friend_repository = friend_repository
        self.transaction_manager = transaction_manager
        self.audit_repository = audit_repository

    async def execute(self, current_user: User, target_id: int) -> MessageResult:
        blocker_id = current_user.require_id()
        if blocker_id == target_id:
            raise ValidationAppError("Нельзя заблокировать себя")
        if not await self.repository.user_exists(target_id):
            raise NotFoundError("User not found")
        async with self.transaction_manager:
            await self.repository.block(blocker_id, target_id)
            await self.friend_repository.remove_friend(blocker_id, target_id)
            await self.friend_repository.unsubscribe(blocker_id, target_id)
            await self.friend_repository.unsubscribe(target_id, blocker_id)
            if self.audit_repository:
                await self.audit_repository.record(
                    blocker_id, "user.blocked", "user", target_id
                )
        return MessageResult(message="Пользователь заблокирован")


class UnblockUserUseCase:
    def __init__(self, repository: BlockRepository, transaction_manager: TransactionManager) -> None:
        self.repository = repository
        self.transaction_manager = transaction_manager

    async def execute(self, current_user: User, target_id: int) -> MessageResult:
        blocker_id = current_user.require_id()
        async with self.transaction_manager:
            removed = await self.repository.unblock(blocker_id, target_id)
        return MessageResult(message="Пользователь разблокирован" if removed else "Пользователь не был заблокирован")
