from fastapi import Response

from backend.infra.config import JwtConfig


def set_access_token_cookie(
    response: Response,
    access_token: str,
    jwt_config: JwtConfig,
) -> None:
    response.set_cookie(
        key="access-token",
        value=access_token,
        httponly=True,
        samesite="lax",
        max_age=jwt_config.access_token_expire_minutes * 60,
        secure=False,
    )


def set_refresh_token_cookie(
    response: Response,
    refresh_token: str,
    jwt_config: JwtConfig,
) -> None:
    response.set_cookie(
        key="refresh-token",
        value=refresh_token,
        httponly=True,
        samesite="lax",
        max_age=jwt_config.refresh_token_expire_days * 24 * 60 * 60,
        secure=False,
    )


def set_auth_cookies(
    response: Response,
    access_token: str,
    refresh_token: str,
    jwt_config: JwtConfig,
) -> None:
    set_access_token_cookie(response, access_token, jwt_config)
    set_refresh_token_cookie(response, refresh_token, jwt_config)
