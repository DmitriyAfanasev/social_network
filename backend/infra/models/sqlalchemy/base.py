from backend.infra.config import CONVENTION
from backend.shared.utils.case_converter import camel_case_to_snake_case
from sqlalchemy import MetaData
from sqlalchemy.orm import DeclarativeBase, Mapped, declared_attr, mapped_column


class Base(DeclarativeBase):
    """Базовый класс для всех моделей."""

    __abstract__ = True
    metadata = MetaData(
        naming_convention=CONVENTION,
    )

    @declared_attr.directive
    def __tablename__(cls) -> str:
        """
        Добавление всем классам наследникам название таблиц названием класса в нижнем регистре.
        """
        return f"{camel_case_to_snake_case(cls.__name__)}s"

    id: Mapped[int] = mapped_column(primary_key=True)
