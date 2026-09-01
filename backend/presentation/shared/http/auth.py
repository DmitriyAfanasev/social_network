from dishka.integrations.fastapi import FromDishka, inject
from fastapi import Cookie, HTTPException, status

from backend.application.exceptions import AuthenticationError
from backend.application.use_cases.auth import GetCurrentUserUseCase
from backend.domain.user.entity import User


@inject
async def get_optional_current_user_from_cookie(
    current_user_use_case: FromDishka[GetCurrentUserUseCase],
    access_token: str | None = Cookie(default=None, alias="access-token"),
) -> User | None:
    return await current_user_use_case.execute(access_token, required=False)


@inject
async def get_current_user_from_cookie(
    current_user_use_case: FromDishka[GetCurrentUserUseCase],
    access_token: str | None = Cookie(default=None, alias="access-token"),
) -> User:
    try:
        user = await current_user_use_case.execute(access_token)
    except AuthenticationError as e:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail=str(e),
        ) from e

    if user is None:
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Пользователь не авторизован")
    return user
