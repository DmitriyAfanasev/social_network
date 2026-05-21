import asyncio
import logging
from contextlib import suppress

from faststream.rabbit import RabbitBroker
from sqlalchemy.ext.asyncio import AsyncEngine, AsyncSession, async_sessionmaker, create_async_engine

from backend.application.use_cases.outbox import PublishOutboxEventsUseCase
from backend.infra.config import DatabaseConfig, EventBusConfig, settings
from backend.infra.messaging.faststream_publisher import FastStreamEventPublisher
from backend.infra.repositories.outbox_repository import OutboxRepository
from backend.infra.transactions.sqlalchemy import SQLAlchemyTransactionManager


logger = logging.getLogger(__name__)


class OutboxPublisherWorker:
    """Фоновый publisher, который переносит события из PostgreSQL outbox в RabbitMQ.

    FastAPI-приложение пишет события только в БД. Этот worker живёт внутри
    FastStream-процесса, периодически читает pending-события и публикует их в
    broker. Это отдельная asyncio task в том же event loop, а не отдельный OS
    thread.
    """

    def __init__(
        self,
        broker: RabbitBroker,
        *,
        database_config: DatabaseConfig = settings.db,
        event_bus_config: EventBusConfig = settings.event_bus,
    ) -> None:
        self.broker = broker
        self.database_config = database_config
        self.event_bus_config = event_bus_config
        self._task: asyncio.Task[None] | None = None
        self._stop_event = asyncio.Event()

    def start(self) -> None:
        """Создать background asyncio task для polling loop.

        `asyncio.create_task` не создаёт новый поток. Он просто ставит корутину
        `_run` в текущий event loop FastStream worker-а, чтобы она выполнялась
        параллельно с consumers.
        """
        if self._task is not None and not self._task.done():
            return

        self._stop_event.clear()
        self._task = asyncio.create_task(self._run())

    async def stop(self) -> None:
        """Попросить polling loop остановиться и дождаться завершения task."""
        self._stop_event.set()
        if self._task is None:
            return

        self._task.cancel()
        with suppress(asyncio.CancelledError):
            await self._task

    async def _run(self) -> None:
        """Запустить polling loop и освободить DB engine при остановке worker-а."""
        engine = self._create_engine()
        session_factory = self._create_session_factory(engine)
        event_publisher = FastStreamEventPublisher(self.broker)

        try:
            while not self._stop_event.is_set():
                await self._run_iteration(
                    session_factory=session_factory,
                    event_publisher=event_publisher,
                )

                await asyncio.sleep(self.event_bus_config.outbox_poll_interval_seconds)
        finally:
            await engine.dispose()

    def _create_engine(self) -> AsyncEngine:
        """Создать отдельный SQLAlchemy engine для worker-процесса.

        Мы не можем переиспользовать FastAPI DI session, потому что FastStream
        worker запускается отдельным процессом и не находится внутри HTTP request
        scope. Поэтому для polling loop создаётся собственный engine.
        """
        return create_async_engine(
            url=str(self.database_config.url),
            echo=self.database_config.echo,
            echo_pool=self.database_config.echo_pool,
            pool_size=self.database_config.pool_size,
            max_overflow=self.database_config.max_overflow,
            pool_pre_ping=True,
        )

    @staticmethod
    def _create_session_factory(engine: AsyncEngine) -> async_sessionmaker[AsyncSession]:
        """Создать factory короткоживущих DB sessions для каждой итерации polling."""
        return async_sessionmaker(
            bind=engine,
            autoflush=False,
            autocommit=False,
            expire_on_commit=False,
        )

    async def _run_iteration(
        self,
        *,
        session_factory: async_sessionmaker[AsyncSession],
        event_publisher: FastStreamEventPublisher,
    ) -> None:
        """Выполнить одну попытку публикации пачки outbox-событий."""
        try:
            async with session_factory() as session:
                use_case = self._create_use_case(
                    session=session,
                    event_publisher=event_publisher,
                )
                published_count = await use_case.execute(
                    limit=self.event_bus_config.outbox_batch_size,
                )
                if published_count:
                    logger.info("Published %s outbox events", published_count)
        except Exception:
            logger.exception("Outbox publisher iteration failed")

    @staticmethod
    def _create_use_case(
        *,
        session: AsyncSession,
        event_publisher: FastStreamEventPublisher,
    ) -> PublishOutboxEventsUseCase:
        """Собрать use case вручную для worker-а вне HTTP DI scope."""
        return PublishOutboxEventsUseCase(
            outbox_repository=OutboxRepository(session=session),
            event_publisher=event_publisher,
            transaction_manager=SQLAlchemyTransactionManager(session=session),
        )
