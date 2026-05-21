import asyncio
from typing import AsyncGenerator

import pytest_asyncio
from sqlalchemy.ext.asyncio import AsyncEngine, AsyncSession, async_sessionmaker, create_async_engine
from sqlalchemy.pool import NullPool

from backend.infra.config import settings


@pytest_asyncio.fixture(scope="class")
async def event_loop() -> AsyncGenerator[asyncio.AbstractEventLoop, None]:
    """Фикстура, предоставляющая цикл событий для тестирования."""
    loop = asyncio.get_event_loop_policy().new_event_loop()
    asyncio.set_event_loop(loop)
    yield loop
    loop.close()


@pytest_asyncio.fixture(scope="class")
async def db_engine_class() -> AsyncGenerator[AsyncEngine, None]:
    """Фикстура, предоставляющая экземпляр `DatabaseHelper`, для использования
    в тестах классов."""
    engine = create_async_engine(
        url=str(settings.db.url),
        echo=settings.db.echo,
        echo_pool=settings.db.echo_pool,
        poolclass=NullPool,
    )
    yield engine
    await engine.dispose()


@pytest_asyncio.fixture(scope="class")
async def db_session(
    db_engine_class: AsyncEngine,
) -> AsyncGenerator[AsyncSession, None]:
    """Фикстура, предоставляющая сессию БД для каждого теста класса."""
    session_factory = async_sessionmaker(
        bind=db_engine_class,
        autoflush=False,
        autocommit=False,
        expire_on_commit=False,
    )
    async with session_factory() as session:
        yield session
