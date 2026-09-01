import asyncio
from collections.abc import AsyncGenerator

import fakeredis
import pytest
import pytest_asyncio
from asgi_lifespan import LifespanManager
from fakeredis.aioredis import FakeRedis
from httpx import ASGITransport, AsyncClient
from sqlalchemy.ext.asyncio import (
    AsyncEngine,
    AsyncSession,
    async_sessionmaker,
    create_async_engine,
)
from sqlalchemy.pool import NullPool

from backend.infra.config import settings
from backend.infra.models.sqlalchemy import Base
from backend.infra.repositories.pending_token_store import RedisPendingTokenStore
from backend.main import main_app


def pytest_collection_modifyitems(items: list[pytest.Item]) -> None:
    """Apply stable layer markers from the test location."""
    for item in items:
        if "/unit/models/" in item.nodeid:
            item.add_marker(pytest.mark.integration)
        elif "/unit/" in item.nodeid:
            item.add_marker(pytest.mark.unit)
        elif "/integration/" in item.nodeid:
            item.add_marker(pytest.mark.integration)


@pytest_asyncio.fixture(scope="function")
async def event_loop() -> AsyncGenerator[asyncio.AbstractEventLoop]:
    """Фикстура, предоставляющая цикл событий для тестирования."""
    print("Запуск цикла событий для уровня тестирования function")
    loop = asyncio.get_event_loop_policy().new_event_loop()
    asyncio.set_event_loop(loop)
    yield loop
    loop.close()


@pytest_asyncio.fixture(scope="session")
async def db_engine() -> AsyncGenerator[AsyncEngine]:
    """Фикстура, предоставляющая экземпляр `DatabaseHelper`, настроенный для
    тестовой БД.

    Использует NullPool для предотвращения проблем с соединениями между
    тестами.
    """
    engine = create_async_engine(
        url=str(settings.db.url),
        echo=settings.db.echo,
        echo_pool=settings.db.echo_pool,
        poolclass=NullPool,
    )
    yield engine
    await engine.dispose()


@pytest_asyncio.fixture(scope="session")
async def setup_test_database(db_engine: AsyncEngine) -> AsyncGenerator[None]:
    """Асинхронная фикстура для настройки тестовой базы данных.

    Перед запуском тестов создаются все таблицы. После выполнения тестов
    — удаляются.
    """
    assert settings.db.MODE == "TEST"
    async with db_engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)

    yield  # тут все действия с базой

    async with db_engine.begin() as conn:
        await conn.run_sync(Base.metadata.drop_all)


@pytest_asyncio.fixture(scope="function")
async def db_session(
    db_engine: AsyncEngine,
) -> AsyncGenerator[AsyncSession]:
    """Фикстура, предоставляющая экземпляр `DatabaseHelper`, настроенный для
    тестовой БД.

    Использует NullPool для предотвращения проблем с соединениями между
    тестами.
    """
    session_factory = async_sessionmaker(
        bind=db_engine,
        autoflush=False,
        autocommit=False,
        expire_on_commit=False,
    )
    async with session_factory() as session:
        yield session


@pytest_asyncio.fixture(scope="function")
async def async_client() -> AsyncGenerator[AsyncClient]:
    """Фикстура, предоставляющая HTTP клиент для тестирования API.

    Использует ASGITransport для тестирования FastAPI приложения.
    """
    async with LifespanManager(main_app), AsyncClient(
        transport=ASGITransport(app=main_app),
        base_url="http://testserver",
    ) as client:
        yield client


@pytest_asyncio.fixture
async def fake_redis_client() -> AsyncGenerator[FakeRedis]:
    """Фикстура, предоставляющая фейковый Redis клиент для тестирования."""
    client = fakeredis.aioredis.FakeRedis(decode_responses=True)
    yield client
    await client.flushall()
    await client.close()


@pytest_asyncio.fixture
async def redis_test_client(
    fake_redis_client: FakeRedis,
) -> AsyncGenerator[RedisPendingTokenStore]:
    """Фикстура, предоставляющая Redis клиент для тестирования."""
    yield RedisPendingTokenStore(
        redis_config=settings.redis,
        redis_instance=fake_redis_client,
    )
