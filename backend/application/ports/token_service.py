from abc import ABC, abstractmethod

from backend.application.dto import AuthTokensDTO


class AuthTokenService(ABC):
    @abstractmethod
    def create_tokens(self, user_id: int) -> AuthTokensDTO:
        pass

    @abstractmethod
    def refresh_access_token(self, refresh_token: str) -> str:
        pass

    @abstractmethod
    def get_access_user_id(self, access_token: str) -> int:
        pass
