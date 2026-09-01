import logging
from collections.abc import AsyncGenerator, Callable
from contextlib import AbstractAsyncContextManager, asynccontextmanager

from dishka.integrations.fastapi import setup_dishka
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from starlette.staticfiles import StaticFiles

from backend.application.exceptions import ApplicationError
from backend.di.container import create_container
from backend.infra.config import BASE_DIR, Settings, settings
from backend.presentation.shared.http.errors import application_error_handler


STATIC_DIR = BASE_DIR / "frontend" / "static"
FRONTEND_DEV_ORIGINS = (
    "http://localhost:5173",
    "http://127.0.0.1:5173",
    "http://localhost:5174",
    "http://127.0.0.1:5174",
)
logger = logging.getLogger(__name__)


def configure_logging(log_format: str) -> None:
    logging.basicConfig(
        level=logging.INFO,
        format=log_format,
    )


def create_lifespan() -> Callable[[FastAPI], AbstractAsyncContextManager[None]]:
    @asynccontextmanager
    async def lifespan(app: FastAPI) -> AsyncGenerator[None]:
        """Контекстный менеджер для управления жизненным циклом приложения.

        Выполняет:
        - Инициализацию ресурсов при старте
        - Корректное освобождение ресурсов при завершении
        """
        logger.info("Начало работы приложения")
        yield
        logger.info("Конец работы приложения")
        await app.state.dishka_container.close()

    return lifespan


def create_app(app_settings: Settings = settings) -> FastAPI:
    """Фабрика для создания экземпляра FastAPI приложения.

    :return: FastAPI: Настроенный экземпляр приложения
    """
    configure_logging(app_settings.logging.log_format)
    application = FastAPI(lifespan=create_lifespan())
    setup_dishka(create_container(), application)
    application.add_middleware(
        CORSMiddleware,
        allow_origins=FRONTEND_DEV_ORIGINS,
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )
    application.add_exception_handler(ApplicationError, application_error_handler)
    application.mount(
        "/static",
        StaticFiles(directory=str(STATIC_DIR)),
        name="static",
    )
    return application
