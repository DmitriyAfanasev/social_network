from collections.abc import Sequence
from typing import Any

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from backend.application.ports.audit_repository import AuditRepository as AuditPort
from backend.infra.models.sqlalchemy import ModerationAuditLog


class AuditRepository(AuditPort):
    def __init__(self, session: AsyncSession) -> None:
        self.session = session

    async def record(self, actor_id: int, action: str, target_type: str, target_id: int | None, details: dict[str, Any] | None = None) -> None:
        self.session.add(ModerationAuditLog(
            actor_id=actor_id,
            action=action,
            target_type=target_type,
            target_id=target_id,
            details=details or {},
        ))
        await self.session.flush()

    async def list(self, limit: int) -> Sequence[Any]:
        result = await self.session.scalars(select(ModerationAuditLog).order_by(
            ModerationAuditLog.created_at.desc(), ModerationAuditLog.id.desc()
        ).limit(limit))
        return list(result.all())
