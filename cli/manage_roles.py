import argparse
import asyncio
from collections.abc import Sequence

from rich.console import Console
from rich.table import Table
from sqlalchemy import select
from sqlalchemy.exc import SQLAlchemyError
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine

from backend.application.exceptions import ApplicationError
from backend.infra.models.sqlalchemy import Role, User
from backend.infra.repositories.admin_repository import AdminRepository


console = Console()


class CliArgumentParser(argparse.ArgumentParser):
    """Argument parser that prints actionable errors before the help text."""

    def error(self, message: str) -> None:  # type: ignore[override]
        console.print("[bold red]Ошибка в аргументах:[/bold red]", message)
        console.print()
        self.print_help()
        raise SystemExit(2)


def _database_url() -> str:
    from backend.infra.config import settings

    return str(settings.db.url)


def build_parser() -> CliArgumentParser:
    """Build the argument parser for role administration."""
    parser = CliArgumentParser(description="Управление ролями пользователей")
    subparsers = parser.add_subparsers(dest="command", required=True)

    role_parser = subparsers.add_parser("grant", help="назначить роль пользователю")
    role_parser.add_argument("--email", required=True, help="email пользователя")
    role_parser.add_argument("--role", required=True, help="имя роли, например admin")

    revoke_parser = subparsers.add_parser("revoke", help="снять роль с пользователя")
    revoke_parser.add_argument("--email", required=True, help="email пользователя")
    revoke_parser.add_argument("--role", required=True, help="имя роли, например admin")

    subparsers.add_parser("list-roles", help="показать доступные роли")
    return parser


async def _find_user(session: AsyncSession, email: str) -> User:
    normalized_email = email.strip().lower()
    if not normalized_email:
        raise ValueError("Email пользователя не может быть пустым")
    user = await session.scalar(select(User).where(User.email == normalized_email))
    if user is None:
        raise ValueError(f"Пользователь с email {normalized_email!r} не найден")
    return user


async def _change_role(command: str, email: str, role_name: str) -> None:
    normalized_role = role_name.strip()
    if not normalized_role:
        raise ValueError("Имя роли не может быть пустым")

    engine = create_async_engine(_database_url(), pool_pre_ping=True)
    session_factory = async_sessionmaker(engine, expire_on_commit=False)

    try:
        async with session_factory() as session:
            async with session.begin():
                user = await _find_user(session, email)
                repository = AdminRepository(session)
                if command == "grant":
                    await repository.assign_role(user.id, normalized_role)
                else:
                    removed = await repository.remove_role(user.id, normalized_role)
                    if not removed:
                        raise ValueError(
                            f"Роль {normalized_role!r} не назначена пользователю {user.email}"
                        )
            action = "назначена" if command == "grant" else "снята"
            console.print(
                f"[bold green]Готово:[/bold green] роль {normalized_role!r} "
                f"{action} пользователю [cyan]{user.email}[/cyan]"
            )
    finally:
        await engine.dispose()


async def _list_roles() -> None:
    engine = create_async_engine(_database_url(), pool_pre_ping=True)
    session_factory = async_sessionmaker(engine, expire_on_commit=False)

    try:
        async with session_factory() as session:
            roles = await session.scalars(select(Role).order_by(Role.name))
            table = Table(title="Доступные роли")
            table.add_column("Роль", style="cyan")
            table.add_column("Описание")
            for role in roles:
                table.add_row(role.name, role.description or "—")
            console.print(table)
    finally:
        await engine.dispose()


async def run(arguments: argparse.Namespace) -> None:
    """Execute the selected administrative command."""
    if arguments.command == "list-roles":
        await _list_roles()
        return
    await _change_role(arguments.command, arguments.email, arguments.role)


def main(argv: Sequence[str] | None = None) -> int:
    """Parse CLI arguments and execute the command."""
    arguments = build_parser().parse_args(argv)
    try:
        asyncio.run(run(arguments))
    except (ApplicationError, SQLAlchemyError, ValueError, RuntimeError, OSError) as error:
        console.print(f"[bold red]Ошибка:[/bold red] {error}")
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
