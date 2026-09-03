from typing import Any

from backend.application.exceptions import PermissionDeniedError, ValidationAppError
from backend.application.ports.admin_repository import AdminRepository
from backend.application.ports.authorization import AuthorizationService
from backend.application.ports.transaction_manager import TransactionManager
from backend.application.results import MessageResult
from backend.domain.user.entity import User


class AdminRbacUseCase:
    def __init__(self, repository: AdminRepository, authorization: AuthorizationService, transaction_manager: TransactionManager) -> None:
        self.repository = repository
        self.authorization = authorization
        self.transaction_manager = transaction_manager

    async def _ensure(self, current_user: User, permission: str) -> int:
        user_id = current_user.require_id()
        if not await self.authorization.has_permission(user_id, permission):
            raise PermissionDeniedError("Недостаточно прав")
        return user_id

    async def list_roles(self, current_user: User) -> list[User]:
        await self._ensure(current_user, "roles.manage")
        return list(await self.repository.list_roles())

    async def assign_role(self, current_user: User, target_id: int, role_name: str) -> MessageResult:
        await self._ensure(current_user, "roles.manage")
        if not role_name.strip():
            raise ValidationAppError("Имя роли не может быть пустым")
        async with self.transaction_manager:
            await self.repository.assign_role(target_id, role_name)
        return MessageResult(message=f"Роль {role_name} назначена пользователю {target_id}")

    async def remove_role(self, current_user: User, target_id: int, role_name: str) -> MessageResult:
        await self._ensure(current_user, "roles.manage")
        async with self.transaction_manager:
            removed = await self.repository.remove_role(target_id, role_name)
        return MessageResult(message="Роль снята" if removed else "Роль не была назначена")

    async def list_audit(self, current_user: User, limit: int) -> list[Any]:
        await self._ensure(current_user, "audit.read")
        if not 1 <= limit <= 200:
            raise ValidationAppError("Параметр limit должен быть от 1 до 200")
        return list(await self.repository.list_audit_logs(limit))
