from datetime import UTC, datetime, timedelta
from typing import Any

import jwt

from backend.application.dto import AuthTokensDTO
from backend.application.ports.token_service import AuthTokenService
from backend.infra.config import JwtConfig


class JwtAuthTokenService(AuthTokenService):
    def __init__(self, jwt_config: JwtConfig) -> None:
        self.jwt_config = jwt_config

    def create_tokens(self, user_id: int) -> AuthTokensDTO:
        return AuthTokensDTO(
            access_token=self._create_token(
                {"user_id": user_id, "type": "access"},
                self._access_ttl,
            ),
            refresh_token=self._create_token(
                {"user_id": user_id, "type": "refresh"},
                self._refresh_ttl,
            ),
        )

    def refresh_access_token(self, refresh_token: str) -> str:
        payload = self._decode(refresh_token)
        if payload.get("type") != "refresh":
            raise ValueError("Invalid token type")

        user_id = self._extract_user_id(payload)
        if user_id is None:
            raise ValueError("Invalid refresh token")
        return self._create_token(
            {"user_id": user_id, "type": "access"},
            self._access_ttl,
        )

    def get_access_user_id(self, access_token: str) -> int:
        payload = self._decode(access_token)
        if payload.get("type") != "access":
            raise ValueError("Invalid token type")

        user_id = self._extract_user_id(payload)
        if user_id is None:
            raise ValueError("Invalid token: missing user_id")
        return user_id

    @property
    def _access_ttl(self) -> timedelta:
        return timedelta(minutes=self.jwt_config.access_token_expire_minutes)

    @property
    def _refresh_ttl(self) -> timedelta:
        return timedelta(days=self.jwt_config.refresh_token_expire_days)

    def _create_token(self, payload: dict[str, Any], ttl: timedelta) -> str:
        if not self.jwt_config.secret_key or not self.jwt_config.algorithm:
            raise ValueError("SECRET_KEY и ALGORITHM должны быть настроены в конфигурации.")

        to_encode = payload.copy()
        expire = datetime.now(UTC) + ttl
        to_encode.update({"exp": int(expire.timestamp())})
        return jwt.encode(
            to_encode,
            self.jwt_config.secret_key.get_secret_value(),
            algorithm=self.jwt_config.algorithm,
        )

    def _decode(self, token: str) -> dict[str, Any]:
        try:
            return jwt.decode(
                token,
                self.jwt_config.secret_key.get_secret_value(),
                algorithms=[self.jwt_config.algorithm],
            )
        except jwt.ExpiredSignatureError as e:
            raise ValueError("Token expired") from e
        except jwt.InvalidTokenError as e:
            raise ValueError("Invalid token") from e

    def _extract_user_id(self, payload: dict[str, Any]) -> int | None:
        user_id = payload.get("user_id")
        if isinstance(user_id, int):
            return user_id
        if isinstance(user_id, str) and user_id.isdecimal():
            return int(user_id)
        return None
