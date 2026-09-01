from sqlalchemy import exists, select
from sqlalchemy.ext.asyncio import AsyncSession

from backend.application.ports.authorization import AuthorizationService as AuthorizationPort
from backend.infra.models.sqlalchemy import Permission, RolePermission, User, UserRole


class AuthorizationService(AuthorizationPort):
    def __init__(self, session: AsyncSession) -> None:
        self.session = session

    async def has_permission(self, user_id: int, permission: str) -> bool:
        role_permission = select(RolePermission.id).where(
            RolePermission.permission_id == Permission.id,
            RolePermission.role_id == UserRole.role_id,
            UserRole.user_id == user_id,
            Permission.code == permission,
        )
        if await self.session.scalar(select(exists(role_permission))):
            return True
        return bool(await self.session.scalar(select(exists().where(
            User.id == user_id,
            User.is_superuser.is_(True),
        ))))
