from collections.abc import Sequence
from typing import Any

from sqlalchemy import delete, select
from sqlalchemy.ext.asyncio import AsyncSession

from backend.application.exceptions import NotFoundError
from backend.application.ports.admin_repository import AdminRepository as AdminPort
from backend.infra.models.sqlalchemy import ModerationAuditLog, Role, UserRole


class AdminRepository(AdminPort):
    def __init__(self, session: AsyncSession) -> None:
        self.session = session

    async def list_roles(self) -> Sequence[Any]:
        result = await self.session.scalars(select(Role).order_by(Role.name))
        return list(result.all())

    async def assign_role(self, user_id: int, role_name: str) -> None:
        role = await self.session.scalar(select(Role).where(Role.name == role_name))
        if role is None:
            raise NotFoundError("Роль не найдена")
        exists = await self.session.scalar(select(UserRole.id).where(
            UserRole.user_id == user_id,
            UserRole.role_id == role.id,
        ))
        if exists is None:
            self.session.add(UserRole(user_id=user_id, role_id=role.id))
            await self.session.flush()

    async def remove_role(self, user_id: int, role_name: str) -> bool:
        role = await self.session.scalar(select(Role).where(Role.name == role_name))
        if role is None:
            raise NotFoundError("Роль не найдена")
        result = await self.session.execute(
            delete(UserRole)
            .where(
                UserRole.user_id == user_id,
                UserRole.role_id == role.id,
            )
            .returning(UserRole.id)
        )
        return bool(result.all())

    async def list_audit_logs(self, limit: int) -> Sequence[Any]:
        result = await self.session.scalars(
            select(ModerationAuditLog).order_by(
                ModerationAuditLog.created_at.desc(), ModerationAuditLog.id.desc()
            ).limit(limit)
        )
        return list(result.all())
