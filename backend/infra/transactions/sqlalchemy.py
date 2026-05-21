from types import TracebackType

from sqlalchemy.ext.asyncio import AsyncSession

from backend.application.ports.transaction_manager import TransactionManager


class SQLAlchemyTransactionManager(TransactionManager):
    def __init__(self, session: AsyncSession) -> None:
        self._session = session

    async def __aenter__(self) -> "SQLAlchemyTransactionManager":
        return self

    async def __aexit__(
        self,
        exc_type: type[BaseException] | None,
        exc: BaseException | None,
        traceback: TracebackType | None,
    ) -> None:
        if exc_type is None:
            await self.commit()
            return
        await self.rollback()

    async def commit(self) -> None:
        await self._session.commit()

    async def rollback(self) -> None:
        await self._session.rollback()
