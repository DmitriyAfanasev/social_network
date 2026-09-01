from dishka import Provider, Scope, provide
from fastapi import HTTPException, Request, status

from backend.application.exceptions import AuthenticationError
from backend.application.use_cases.auth import GetCurrentUserUseCase
from backend.domain.user.entity import User


class AuthProvider(Provider):
    @provide(scope=Scope.REQUEST)
    async def current_user(
        self,
        request: Request,
        current_user_use_case: GetCurrentUserUseCase,
    ) -> User:
        token = request.cookies.get("access-token")
        try:
            user = await current_user_use_case.execute(token)
        except AuthenticationError as error:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail=str(error),
            ) from error
        if user is None or user.id is None:
            raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Invalid user")
        return user
