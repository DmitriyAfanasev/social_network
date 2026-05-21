from abc import ABC, abstractmethod
from types import TracebackType


class TransactionManager(ABC):
    @abstractmethod
    async def __aenter__(self) -> "TransactionManager":
        pass

    @abstractmethod
    async def __aexit__(
        self,
        exc_type: type[BaseException] | None,
        exc: BaseException | None,
        traceback: TracebackType | None,
    ) -> None:
        pass

    @abstractmethod
    async def commit(self) -> None:
        pass

    @abstractmethod
    async def rollback(self) -> None:
        pass
