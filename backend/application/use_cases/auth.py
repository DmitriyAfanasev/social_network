import logging
import uuid

from backend.application.analytics_events import build_analytics_event_payload
from backend.application.commands import (
    ConfirmRegistrationCommand,
    LoginCommand,
    PasswordResetCommand,
    PasswordResetRequestCommand,
    RegisterUserCommand,
    RegistrationConfirmationCommand,
)
from backend.application.event_types import USER_REGISTERED_EVENT
from backend.application.events import IntegrationEvent
from backend.application.exceptions import (
    ApplicationError,
    AuthenticationError,
    ExternalServiceError,
    NotFoundError,
    ValidationAppError,
)
from backend.application.ports.notification_sender import NotificationSender
from backend.application.ports.outbox_repository import OutboxRepository
from backend.application.ports.password_hasher import PasswordHasher
from backend.application.ports.token_service import AuthTokenService
from backend.application.ports.token_store import PendingTokenStore
from backend.application.ports.transaction_manager import TransactionManager
from backend.application.ports.user_repository import UserRepository
from backend.application.results import AccessTokenResult, AuthResult, MessageResult
from backend.domain.user.entity import User


logger = logging.getLogger(__name__)

NOT_FOUND_USER_MESSAGE = "Пользователь с таким email не найден"
INVALID_TOKEN_MESSAGE = "Невалидный токен"


def _exception_message(error: Exception) -> str:
    return str(getattr(error, "detail", error))


def _persisted_user_id(user: User) -> int:
    if user.id is None:
        raise ExternalServiceError("У пользователя отсутствует ID")
    return user.id


class LoginUseCase:
    def __init__(
        self,
        user_repository: UserRepository,
        password_hasher: PasswordHasher,
        token_service: AuthTokenService,
    ) -> None:
        self.user_repository = user_repository
        self.password_hasher = password_hasher
        self.token_service = token_service

    async def execute(self, command: LoginCommand) -> AuthResult:
        try:
            user = await self.user_repository.get_by_email(str(command.email))
            if user is None:
                raise AuthenticationError("Неверный email или пароль.")
            if not self.password_hasher.verify(command.password, user.hashed_password):
                raise AuthenticationError("Неверный email или пароль.")
            if not user.is_active:
                raise AuthenticationError(
                    "Учетная запись заблокирована, либо слишком давно не была активна."
                )
        except AuthenticationError:
            raise
        except Exception as e:
            raise AuthenticationError(_exception_message(e)) from e
        return AuthResult(
            user=user,
            tokens=self.token_service.create_tokens(_persisted_user_id(user)),
        )


class RefreshTokenUseCase:
    def __init__(self, token_service: AuthTokenService) -> None:
        self.token_service = token_service

    async def execute(self, refresh_token: str | None) -> AccessTokenResult:
        if not refresh_token:
            raise AuthenticationError("Refresh token is missing")
        try:
            return AccessTokenResult(
                access_token=self.token_service.refresh_access_token(refresh_token)
            )
        except Exception as e:
            raise AuthenticationError(_exception_message(e)) from e


class RequestRegistrationConfirmationUseCase:
    def __init__(
        self,
        user_repository: UserRepository,
        token_store: PendingTokenStore,
        notification_sender: NotificationSender,
    ) -> None:
        self.user_repository = user_repository
        self.token_store = token_store
        self.notification_sender = notification_sender

    async def execute(self, command: RegistrationConfirmationCommand) -> MessageResult:
        try:
            user = await self.user_repository.get_by_email(str(command.email))
            if user is None:
                raise NotFoundError("Пользователь с таким email не найден")
            if user.is_active:
                raise ValidationAppError("Email уже подтвержден")

            temporary_user_token = str(uuid.uuid4())
            saved = await self.token_store.save_registration_confirmation_token(
                temporary_user_token,
                command.email,
            )
            if not saved:
                raise ExternalServiceError("Не удалось сохранить токен подтверждения")
            await self.notification_sender.send_registration_confirmation(
                command.email,
                temporary_user_token,
                command.base_url,
            )
            return MessageResult("Confirmation email sent")
        except ApplicationError:
            raise
        except Exception as e:
            logger.error("Registration confirmation failed: %s", e, exc_info=True)
            raise ExternalServiceError("Произошла ошибка при регистрации") from e


class RegisterUserUseCase:
    def __init__(
        self,
        user_repository: UserRepository,
        password_hasher: PasswordHasher,
        token_store: PendingTokenStore,
        notification_sender: NotificationSender,
        outbox_repository: OutboxRepository,
        transaction_manager: TransactionManager,
    ) -> None:
        self.user_repository = user_repository
        self.password_hasher = password_hasher
        self.token_store = token_store
        self.notification_sender = notification_sender
        self.outbox_repository = outbox_repository
        self.transaction_manager = transaction_manager

    async def execute(self, command: RegisterUserCommand) -> MessageResult:
        if command.password != command.password2:
            raise ValidationAppError("Пароли не совпадают")

        try:
            async with self.transaction_manager:
                if await self.user_repository.get_by_email(str(command.email)):
                    raise ValidationAppError(
                        f"Пользователь с email {command.email!r} уже существует."
                    )
                if await self.user_repository.get_by_username(command.username):
                    raise ValidationAppError(
                        f"Пользователь с username {command.username!r} уже существует."
                    )

                user = User(
                    username=command.username,
                    email=str(command.email),
                    hashed_password=self.password_hasher.hash(command.password),
                    is_active=False,
                )
                user = await self.user_repository.create(user)
                await self.outbox_repository.add(
                    IntegrationEvent(
                        event_type=USER_REGISTERED_EVENT,
                        payload=build_analytics_event_payload(
                            user_id=_persisted_user_id(user),
                            entity_type="user",
                            entity_id=_persisted_user_id(user),
                            data={},
                        ),
                    )
                )
            confirmation_token = str(uuid.uuid4())
            saved = await self.token_store.save_registration_confirmation_token(
                confirmation_token,
                user.email,
            )
            if not saved:
                raise ExternalServiceError("Не удалось сохранить токен подтверждения")
            await self.notification_sender.send_registration_confirmation(
                user.email,
                confirmation_token,
                command.base_url,
            )
            return MessageResult("Confirmation email sent")
        except ValueError as e:
            raise ValidationAppError(str(e.args[0])) from e
        except ApplicationError:
            raise
        except Exception as e:
            logger.error("Registration failed: %s", e, exc_info=True)
            raise ExternalServiceError("Произошла ошибка при регистрации") from e


class ConfirmRegistrationUseCase:
    def __init__(
        self,
        user_repository: UserRepository,
        token_store: PendingTokenStore,
        token_service: AuthTokenService,
        transaction_manager: TransactionManager,
    ) -> None:
        self.user_repository = user_repository
        self.token_store = token_store
        self.token_service = token_service
        self.transaction_manager = transaction_manager

    async def execute(self, command: ConfirmRegistrationCommand) -> AuthResult:
        email = await self.token_store.get_email_by_token(command.token)
        if email is None:
            raise ValidationAppError(INVALID_TOKEN_MESSAGE)

        async with self.transaction_manager:
            user = await self.user_repository.activate_by_email(email)
            if user is None:
                raise ValidationAppError(NOT_FOUND_USER_MESSAGE)
            await self.token_store.delete_token(command.token)

        return AuthResult(
            user=user,
            tokens=self.token_service.create_tokens(_persisted_user_id(user)),
        )


class RequestPasswordResetUseCase:
    def __init__(
        self,
        user_repository: UserRepository,
        token_store: PendingTokenStore,
        notification_sender: NotificationSender,
    ) -> None:
        self.user_repository = user_repository
        self.token_store = token_store
        self.notification_sender = notification_sender

    async def execute(self, command: PasswordResetRequestCommand) -> MessageResult:
        try:
            user = await self.user_repository.get_by_email(str(command.email))
            if user is None:
                raise NotFoundError("Пользователя с таким Email не существует.")
            reset_token = str(uuid.uuid4())
            saved = await self.token_store.save_password_reset_token(
                reset_token,
                command.email,
            )
            if not saved:
                raise ExternalServiceError("Не удалось сохранить токен сброса пароля")
            await self.notification_sender.send_password_reset(
                command.email,
                reset_token,
                command.base_url,
            )
            return MessageResult("Password reset email sent")
        except ApplicationError:
            raise
        except Exception as e:
            logger.error("Password reset request failed: %s", e)
            raise ExternalServiceError("Ошибка при запросе сброса пароля") from e


class ResetPasswordUseCase:
    def __init__(
        self,
        user_repository: UserRepository,
        token_store: PendingTokenStore,
        password_hasher: PasswordHasher,
        token_service: AuthTokenService,
        transaction_manager: TransactionManager,
    ) -> None:
        self.user_repository = user_repository
        self.token_store = token_store
        self.password_hasher = password_hasher
        self.token_service = token_service
        self.transaction_manager = transaction_manager

    async def execute(self, command: PasswordResetCommand) -> AuthResult:
        if command.new_password != command.confirm_password:
            raise ValidationAppError("Пароли не совпадают")

        try:
            try:
                email = await self.token_store.get_email_by_token(command.token)
            except Exception as e:
                logger.error("Password reset token lookup failed: %s", e)
                raise ExternalServiceError("Ошибка при смене пароля") from e
            if email is None:
                raise ValidationAppError(INVALID_TOKEN_MESSAGE)
            user = await self.user_repository.get_by_email(email)
            if user is None:
                raise ValidationAppError(NOT_FOUND_USER_MESSAGE)
            async with self.transaction_manager:
                updated_user = await self.user_repository.change_password_by_email(
                    email,
                    self.password_hasher.hash(command.new_password),
                )
            if updated_user is None:
                raise ValidationAppError(NOT_FOUND_USER_MESSAGE)
            return AuthResult(
                user=updated_user,
                tokens=self.token_service.create_tokens(_persisted_user_id(updated_user)),
            )
        except ApplicationError:
            raise
        except Exception as e:
            if hasattr(e, "detail"):
                raise ValidationAppError(_exception_message(e)) from e
            logger.error("Password reset failed: %s", e)
            raise ExternalServiceError("Ошибка при смене пароля") from e
