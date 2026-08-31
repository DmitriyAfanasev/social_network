from backend.domain.user.entity import User


def require_user_id(user: User) -> int:
    if user.id is None:
        raise RuntimeError("Authenticated user must have an id")
    return user.id
