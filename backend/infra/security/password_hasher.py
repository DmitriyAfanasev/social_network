from backend.application.ports.password_hasher import PasswordHasher
from backend.infra.security.password_helper import PasswordHelper


class BcryptPasswordHasher(PasswordHasher):
    def hash(self, password: str) -> str:
        return PasswordHelper.generate_password(password)

    def verify(self, plain_password: str, hashed_password: str) -> bool:
        return PasswordHelper.verify_password(plain_password, hashed_password)
