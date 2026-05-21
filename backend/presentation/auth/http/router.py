from dishka.integrations.fastapi import DishkaRoute, FromDishka
from fastapi import APIRouter, Cookie, Response

from backend.application.commands import (
    ConfirmRegistrationCommand,
    LoginCommand,
    PasswordResetCommand,
    PasswordResetRequestCommand,
    RegisterUserCommand,
    RegistrationConfirmationCommand,
)
from backend.application.use_cases.auth import (
    ConfirmRegistrationUseCase,
    LoginUseCase,
    RefreshTokenUseCase,
    RegisterUserUseCase,
    RequestPasswordResetUseCase,
    RequestRegistrationConfirmationUseCase,
    ResetPasswordUseCase,
)
from backend.infra.config import FrontendConfig, JwtConfig
from backend.presentation.auth.http.schemas import (
    LoginRequest,
    PasswordResetConfirmRequest,
    PasswordResetRequest,
    RegisterUserRequest,
    RegistrationConfirmationConfirmRequest,
    RegistrationConfirmationRequest,
)
from backend.presentation.shared.http.cookies import (
    set_access_token_cookie,
    set_auth_cookies,
)
from backend.presentation.shared.http.schemas import AuthResponse, MessageResponse
from backend.presentation.shared.http.serializers import (
    auth_result_to_response,
    message_result_to_response,
)


router = APIRouter(tags=["Auth"], route_class=DishkaRoute)


@router.post("/login", response_model=AuthResponse)
async def login(
    body: LoginRequest,
    response: Response,
    use_case: FromDishka[LoginUseCase],
    jwt_config: FromDishka[JwtConfig],
) -> AuthResponse:
    result = await use_case.execute(
        LoginCommand(
            email=body.email,
            password=body.password,
        )
    )
    set_auth_cookies(
        response,
        result.tokens.access_token,
        result.tokens.refresh_token,
        jwt_config,
    )
    return auth_result_to_response(result)


@router.post("/refresh", response_model=MessageResponse)
async def refresh_token(
    response: Response,
    use_case: FromDishka[RefreshTokenUseCase],
    jwt_config: FromDishka[JwtConfig],
    refresh_token: str | None = Cookie(default=None, alias="refresh-token"),
) -> MessageResponse:
    result = await use_case.execute(refresh_token)
    set_access_token_cookie(response, result.access_token, jwt_config)
    return MessageResponse(message="Access token refreshed")


@router.post("/logout", response_model=MessageResponse)
async def logout(response: Response) -> MessageResponse:
    response.delete_cookie("access-token")
    response.delete_cookie("refresh-token")
    return MessageResponse(message="Logged out")


@router.post("/registration-confirmations", response_model=MessageResponse)
async def request_registration_confirmation(
    body: RegistrationConfirmationRequest,
    use_case: FromDishka[RequestRegistrationConfirmationUseCase],
    frontend_config: FromDishka[FrontendConfig],
) -> MessageResponse:
    result = await use_case.execute(
        RegistrationConfirmationCommand(
            email=body.email,
            base_url=frontend_config.base_url,
        )
    )
    return message_result_to_response(result)


@router.post("/register", response_model=MessageResponse)
async def register_user(
    body: RegisterUserRequest,
    use_case: FromDishka[RegisterUserUseCase],
    frontend_config: FromDishka[FrontendConfig],
) -> MessageResponse:
    result = await use_case.execute(
        RegisterUserCommand(
            **body.model_dump(),
            base_url=frontend_config.base_url,
        )
    )
    return message_result_to_response(result)


@router.post("/registration-confirmations/confirm", response_model=AuthResponse)
async def confirm_registration(
    body: RegistrationConfirmationConfirmRequest,
    response: Response,
    use_case: FromDishka[ConfirmRegistrationUseCase],
    jwt_config: FromDishka[JwtConfig],
) -> AuthResponse:
    result = await use_case.execute(ConfirmRegistrationCommand(**body.model_dump()))
    set_auth_cookies(
        response,
        result.tokens.access_token,
        result.tokens.refresh_token,
        jwt_config,
    )
    return auth_result_to_response(result)


@router.post("/password-reset-requests", response_model=MessageResponse)
async def request_password_reset(
    body: PasswordResetRequest,
    use_case: FromDishka[RequestPasswordResetUseCase],
    frontend_config: FromDishka[FrontendConfig],
) -> MessageResponse:
    result = await use_case.execute(
        PasswordResetRequestCommand(
            email=body.email,
            base_url=frontend_config.base_url,
        )
    )
    return message_result_to_response(result)


@router.post("/password-resets", response_model=AuthResponse)
async def reset_password(
    body: PasswordResetConfirmRequest,
    response: Response,
    use_case: FromDishka[ResetPasswordUseCase],
    jwt_config: FromDishka[JwtConfig],
) -> AuthResponse:
    result = await use_case.execute(PasswordResetCommand(**body.model_dump()))
    set_auth_cookies(
        response,
        result.tokens.access_token,
        result.tokens.refresh_token,
        jwt_config,
    )
    return auth_result_to_response(result)
