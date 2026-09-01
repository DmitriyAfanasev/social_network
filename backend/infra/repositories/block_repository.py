from sqlalchemy import delete, exists, or_, select
from sqlalchemy.ext.asyncio import AsyncSession

from backend.application.ports.block_repository import BlockRepository as BlockPort
from backend.infra.models.sqlalchemy import User, UserBlock


class BlockRepository(BlockPort):
    def __init__(self, session: AsyncSession) -> None:
        self.session = session

    async def user_exists(self, user_id: int) -> bool:
        return bool(await self.session.scalar(select(exists().where(User.id == user_id))))

    async def is_blocked(self, user_id: int, other_user_id: int) -> bool:
        statement = select(exists().where(
            or_(
                (UserBlock.blocker_id == user_id) & (UserBlock.blocked_id == other_user_id),
                (UserBlock.blocker_id == other_user_id) & (UserBlock.blocked_id == user_id),
            )
        ))
        return bool(await self.session.scalar(statement))

    async def block(self, blocker_id: int, blocked_id: int) -> None:
        if await self._direct_block_exists(blocker_id, blocked_id):
            return
        self.session.add(UserBlock(blocker_id=blocker_id, blocked_id=blocked_id))
        await self.session.flush()

    async def unblock(self, blocker_id: int, blocked_id: int) -> bool:
        if not await self._direct_block_exists(blocker_id, blocked_id):
            return False
        await self.session.execute(delete(UserBlock).where(
            UserBlock.blocker_id == blocker_id,
            UserBlock.blocked_id == blocked_id,
        ))
        return True

    async def _direct_block_exists(self, blocker_id: int, blocked_id: int) -> bool:
        return bool(await self.session.scalar(select(exists().where(
            UserBlock.blocker_id == blocker_id,
            UserBlock.blocked_id == blocked_id,
        ))))
