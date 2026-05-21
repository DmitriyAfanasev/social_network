from dishka.integrations.fastapi import FromDishka, inject
from fastapi import Cookie, HTTPException, status

from backend.application.ports.token_service import AuthTokenService
from backend.application.ports.user_repository import UserRepository
from backend.domain.user.entity import User


@inject
async def get_optional_current_user_from_cookie(
    token_service: FromDishka[AuthTokenService],
    user_repository: FromDishka[UserRepository],
    access_token: str | None = Cookie(default=None, alias="access-token"),
) -> User | None:
    if not access_token:
        return None

    try:
        user_id = token_service.get_access_user_id(access_token)
    except ValueError:
        return None

    user = await user_repository.get_by_id(user_id)
    if user is None or not user.is_active:
        return None
    return user


@inject
async def get_current_user_from_cookie(
    token_service: FromDishka[AuthTokenService],
    user_repository: FromDishka[UserRepository],
    access_token: str | None = Cookie(default=None, alias="access-token"),
) -> User:
    if not access_token:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Пользователь не авторизован",
        )

    try:
        user_id = token_service.get_access_user_id(access_token)
    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail=str(e),
        ) from e

    user = await user_repository.get_by_id(user_id)
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="User not found",
        )
    if not user.is_active:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="User is inactive",
        )
    return user
