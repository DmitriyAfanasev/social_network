import re


def validate_username(username: str) -> str:
    if not re.match(r"^[a-zA-Z0-9_]+$", username):
        raise ValueError("Username can only contain letters, numbers and underscores")
    return username
