from types import TracebackType

from backend.application.ports.transaction_manager import TransactionManager


class FakeTransaction(TransactionManager):
    """In-memory transaction double implementing the application port."""

    async def __aenter__(self) -> "FakeTransaction":
        return self

    async def __aexit__(
        self,
        exc_type: type[BaseException] | None,
        exc: BaseException | None,
        traceback: TracebackType | None,
    ) -> None:
        return None

    async def commit(self) -> None:
        return None

    async def rollback(self) -> None:
        return None
