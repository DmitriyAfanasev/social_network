from dishka import Provider, Scope, provide
from fastapi import HTTPException, Request, status

from backend.application.ports.token_service import AuthTokenService
from backend.application.ports.user_repository import UserRepository
from backend.domain.user.entity import User


class AuthProvider(Provider):
    @provide(scope=Scope.REQUEST)
    async def current_user(
        self,
        request: Request,
        token_service: AuthTokenService,
        user_repository: UserRepository,
    ) -> User:
        token = request.cookies.get("access-token")
        if token is None:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="access-token is missing in cookie",
            )

        try:
            user_id = token_service.get_access_user_id(token)
        except ValueError as error:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail=str(error),
            ) from error

        user = await user_repository.get_by_id(user_id)
        if user is None:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="User not found",
            )
        if not user.is_active:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="User is inactive",
            )
        if user.id is None:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Invalid user: missing id",
            )
        return user
