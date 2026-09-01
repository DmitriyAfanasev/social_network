from typing import Any
from unittest.mock import AsyncMock

import pytest

from backend.application.exceptions import PermissionDeniedError, ValidationAppError
from backend.application.use_cases.admin import AdminRbacUseCase
from backend.domain.user.entity import User
from tests.fakes import FakeTransaction


def make_user(user_id: int = 1) -> User:
    return User(
        id=user_id,
        username=f"user{user_id}",
        email=f"user{user_id}@example.com",
        hashed_password="hash",
    )


@pytest.mark.application
@pytest.mark.asyncio
@pytest.mark.parametrize(
    ("method", "args", "permission"),
    [
        ("list_roles", (), "roles.manage"),
        ("assign_role", (2, "moderator"), "roles.manage"),
        ("remove_role", (2, "moderator"), "roles.manage"),
        ("list_audit", (50,), "audit.read"),
    ],
)
async def test_admin_actions_require_their_permission(
    method: str,
    args: tuple[Any, ...],
    permission: str,
) -> None:
    repository = AsyncMock()
    authorization = AsyncMock()
    authorization.has_permission.return_value = False
    use_case = AdminRbacUseCase(repository, authorization, FakeTransaction())

    with pytest.raises(PermissionDeniedError):
        await getattr(use_case, method)(make_user(), *args)

    authorization.has_permission.assert_awaited_once_with(1, permission)
    repository.assign_role.assert_not_awaited()
    repository.remove_role.assert_not_awaited()


@pytest.mark.application
@pytest.mark.asyncio
async def test_admin_can_assign_role_after_permission_check() -> None:
    repository = AsyncMock()
    authorization = AsyncMock()
    authorization.has_permission.return_value = True

    result = await AdminRbacUseCase(
        repository, authorization, FakeTransaction()
    ).assign_role(make_user(), 2, "moderator")

    assert result.message == "Роль moderator назначена пользователю 2"
    repository.assign_role.assert_awaited_once_with(2, "moderator")


@pytest.mark.application
@pytest.mark.asyncio
async def test_admin_cannot_assign_blank_role() -> None:
    authorization = AsyncMock()
    authorization.has_permission.return_value = True
    repository = AsyncMock()

    with pytest.raises(ValidationAppError, match="Имя роли"):
        await AdminRbacUseCase(
            repository, authorization, FakeTransaction()
        ).assign_role(make_user(), 2, "   ")

    repository.assign_role.assert_not_awaited()


@pytest.mark.application
@pytest.mark.asyncio
async def test_admin_lists_roles_and_audit_only_after_permission_check() -> None:
    repository = AsyncMock()
    repository.list_roles.return_value = ["moderator"]
    repository.list_audit_logs.return_value = ["entry"]
    authorization = AsyncMock()
    authorization.has_permission.return_value = True
    use_case = AdminRbacUseCase(repository, authorization, FakeTransaction())

    assert await use_case.list_roles(make_user()) == ["moderator"]
    assert await use_case.list_audit(make_user(), 10) == ["entry"]
    repository.list_roles.assert_awaited_once_with()
    repository.list_audit_logs.assert_awaited_once_with(10)

    with pytest.raises(ValidationAppError):
        await use_case.list_audit(make_user(), 201)
    repository.list_audit_logs.assert_awaited_once_with(10)


@pytest.mark.application
@pytest.mark.asyncio
@pytest.mark.parametrize("removed", [True, False])
async def test_admin_remove_role_reports_repository_result(removed: bool) -> None:
    repository = AsyncMock()
    repository.remove_role.return_value = removed
    authorization = AsyncMock()
    authorization.has_permission.return_value = True

    result = await AdminRbacUseCase(
        repository, authorization, FakeTransaction()
    ).remove_role(make_user(), 2, "moderator")

    assert result.message == ("Роль снята" if removed else "Роль не была назначена")
