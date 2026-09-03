from unittest.mock import AsyncMock, Mock

import pytest

from backend.application.commands import (
    ConfirmRegistrationCommand,
    LoginCommand,
    PasswordResetCommand,
    PasswordResetRequestCommand,
    RegisterUserCommand,
    RegistrationConfirmationCommand,
)
from backend.application.dto import AuthTokensDTO
from backend.application.exceptions import AuthenticationError, ValidationAppError
from backend.application.use_cases.auth import (
    ConfirmRegistrationUseCase,
    LoginUseCase,
    RefreshTokenUseCase,
    RegisterUserUseCase,
    RequestPasswordResetUseCase,
    RequestRegistrationConfirmationUseCase,
    ResetPasswordUseCase,
)
from backend.domain.user.entity import User
from tests.fakes import FakeTransaction


def make_user(*, active: bool = True) -> User:
    return User(
        id=7,
        username="alice",
        email="alice@example.com",
        hashed_password="hashed-password",
        is_active=active,
    )


def registration_command() -> RegisterUserCommand:
    return RegisterUserCommand(
        username="alice",
        email="alice@example.com",
        password="secret",
        password2="secret",
        base_url="http://example.com",
    )


@pytest.mark.application
@pytest.mark.asyncio
async def test_login_verifies_password_and_issues_tokens() -> None:
    repository = Mock()
    repository.get_by_email = AsyncMock(return_value=make_user())
    password_hasher = Mock()
    password_hasher.verify.return_value = True
    token_service = Mock()
    token_service.create_tokens.return_value = AuthTokensDTO("access", "refresh")

    result = await LoginUseCase(repository, password_hasher, token_service).execute(
        LoginCommand(email="alice@example.com", password="secret")
    )

    assert result.user.id == 7
    assert result.tokens == AuthTokensDTO("access", "refresh")
    repository.get_by_email.assert_awaited_once_with("alice@example.com")
    password_hasher.verify.assert_called_once_with("secret", "hashed-password")
    token_service.create_tokens.assert_called_once_with(7)


@pytest.mark.application
@pytest.mark.asyncio
@pytest.mark.parametrize(
    ("user", "password_matches"),
    [(None, True), (make_user(), False), (make_user(active=False), True)],
)
async def test_login_rejects_invalid_credentials_or_inactive_account(
    user: User | None,
    password_matches: bool,
) -> None:
    repository = Mock()
    repository.get_by_email = AsyncMock(return_value=user)
    password_hasher = Mock()
    password_hasher.verify.return_value = password_matches
    token_service = Mock()

    with pytest.raises(AuthenticationError):
        await LoginUseCase(repository, password_hasher, token_service).execute(
            LoginCommand(email="alice@example.com", password="wrong")
        )

    token_service.create_tokens.assert_not_called()


@pytest.mark.application
@pytest.mark.asyncio
async def test_refresh_token_returns_new_access_token() -> None:
    token_service = Mock()
    token_service.refresh_access_token.return_value = "new-access"

    result = await RefreshTokenUseCase(token_service).execute("refresh")

    assert result.access_token == "new-access"
    token_service.refresh_access_token.assert_called_once_with("refresh")


@pytest.mark.application
@pytest.mark.asyncio
@pytest.mark.parametrize("refresh_token", [None, ""])
async def test_refresh_token_rejects_missing_token(refresh_token: str | None) -> None:
    with pytest.raises(AuthenticationError):
        await RefreshTokenUseCase(Mock()).execute(refresh_token)


@pytest.mark.application
@pytest.mark.asyncio
async def test_registration_confirmation_saves_token_and_sends_email() -> None:
    repository = Mock()
    repository.get_by_email = AsyncMock(return_value=make_user(active=False))
    token_store = Mock()
    token_store.save_registration_confirmation_token = AsyncMock(return_value=True)
    notification = Mock()
    notification.send_registration_confirmation = AsyncMock()

    result = await RequestRegistrationConfirmationUseCase(
        repository, token_store, notification
    ).execute(RegistrationConfirmationCommand("alice@example.com", "http://example.com"))

    assert result.message == "Confirmation email sent"
    token_store.save_registration_confirmation_token.assert_awaited_once()
    notification.send_registration_confirmation.assert_awaited_once()


@pytest.mark.application
@pytest.mark.asyncio
async def test_registration_confirmation_rejects_active_user() -> None:
    repository = Mock()
    repository.get_by_email = AsyncMock(return_value=make_user())

    with pytest.raises(ValidationAppError, match="Email уже подтвержден"):
        await RequestRegistrationConfirmationUseCase(
            repository, Mock(), Mock()
    ).execute(RegistrationConfirmationCommand("alice@example.com", "http://example.com"))


@pytest.mark.application
@pytest.mark.asyncio
async def test_register_user_creates_inactive_user_and_sends_confirmation() -> None:
    repository = Mock()
    repository.get_by_email = AsyncMock(return_value=None)
    repository.get_by_username = AsyncMock(return_value=None)
    repository.create = AsyncMock(return_value=make_user(active=False))
    hasher = Mock()
    hasher.hash.return_value = "hashed"
    token_store = Mock()
    token_store.save_registration_confirmation_token = AsyncMock(return_value=True)
    notification = Mock()
    notification.send_registration_confirmation = AsyncMock()
    outbox = Mock()
    outbox.add = AsyncMock()

    result = await RegisterUserUseCase(
        repository, hasher, token_store, notification, outbox, FakeTransaction()
    ).execute(registration_command())

    assert result.message == "Confirmation email sent"
    created = repository.create.await_args.args[0]
    assert created.is_active is False
    assert created.hashed_password == "hashed"
    outbox.add.assert_awaited_once()
    notification.send_registration_confirmation.assert_awaited_once()


@pytest.mark.application
@pytest.mark.asyncio
async def test_register_user_rejects_mismatched_passwords_before_dependencies() -> None:
    command = RegisterUserCommand(
        username="alice", email="alice@example.com", password="one", password2="two", base_url="http://example.com"
    )

    with pytest.raises(ValidationAppError, match="Пароли не совпадают"):
        await RegisterUserUseCase(
            Mock(), Mock(), Mock(), Mock(), Mock(), FakeTransaction()
        ).execute(command)


@pytest.mark.application
@pytest.mark.asyncio
async def test_confirm_registration_activates_user_deletes_token_and_issues_tokens() -> None:
    repository = Mock()
    repository.activate_by_email = AsyncMock(return_value=make_user())
    token_store = Mock()
    token_store.get_email_by_token = AsyncMock(return_value="alice@example.com")
    token_store.delete_token = AsyncMock()
    token_service = Mock()
    token_service.create_tokens.return_value = AuthTokensDTO("access", "refresh")

    result = await ConfirmRegistrationUseCase(
        repository, token_store, token_service, FakeTransaction()
    ).execute(ConfirmRegistrationCommand("confirmation"))

    assert result.tokens.access_token == "access"
    repository.activate_by_email.assert_awaited_once_with("alice@example.com")
    token_store.delete_token.assert_awaited_once_with("confirmation")


@pytest.mark.application
@pytest.mark.asyncio
async def test_request_password_reset_saves_token_and_sends_email() -> None:
    repository = Mock()
    repository.get_by_email = AsyncMock(return_value=make_user())
    token_store = Mock()
    token_store.save_password_reset_token = AsyncMock(return_value=True)
    notification = Mock()
    notification.send_password_reset = AsyncMock()

    result = await RequestPasswordResetUseCase(
        repository, token_store, notification
    ).execute(PasswordResetRequestCommand("alice@example.com", "http://example.com"))

    assert result.message == "Password reset email sent"
    token_store.save_password_reset_token.assert_awaited_once()
    notification.send_password_reset.assert_awaited_once()


@pytest.mark.application
@pytest.mark.asyncio
async def test_reset_password_changes_hash_and_issues_tokens() -> None:
    repository = Mock()
    repository.get_by_email = AsyncMock(return_value=make_user())
    repository.change_password_by_email = AsyncMock(return_value=make_user())
    token_store = Mock()
    token_store.get_email_by_token = AsyncMock(return_value="alice@example.com")
    hasher = Mock()
    hasher.hash.return_value = "new-hash"
    token_service = Mock()
    token_service.create_tokens.return_value = AuthTokensDTO("access", "refresh")

    result = await ResetPasswordUseCase(
        repository, token_store, hasher, token_service, FakeTransaction()
    ).execute(PasswordResetCommand("reset", "new-secret", "new-secret"))

    assert result.tokens.refresh_token == "refresh"
    repository.change_password_by_email.assert_awaited_once_with(
        "alice@example.com", "new-hash"
    )


@pytest.mark.application
@pytest.mark.asyncio
async def test_reset_password_rejects_mismatched_passwords_before_token_lookup() -> None:
    token_store = Mock()

    with pytest.raises(ValidationAppError, match="Пароли не совпадают"):
        await ResetPasswordUseCase(
            Mock(), token_store, Mock(), Mock(), FakeTransaction()
        ).execute(PasswordResetCommand("reset", "one", "two"))

    token_store.get_email_by_token.assert_not_called()
