from datetime import datetime
from typing import Annotated

from annotated_types import MaxLen, MinLen
from pydantic import BaseModel, EmailStr, Field
from pydantic.functional_validators import AfterValidator

from backend.presentation.profiles.http.schemas import ProfileCreateRequest
from backend.shared.utils.validated import validate_username


class UserBaseRequest(BaseModel):
    username: Annotated[
        str,
        MinLen(3),
        MaxLen(20),
        AfterValidator(validate_username),
    ] = Field(
        description="Username must be between 3 and 20 characters, only letters, numbers, underscores"
    )
    email: Annotated[EmailStr, MaxLen(100)] = Field(
        description="Valid email address, max 100 chars"
    )


class UserCreateRequest(UserBaseRequest):
    password: Annotated[str, MinLen(8)] = Field(
        description="Password must contain at least 8 characters"
    )
    profile: ProfileCreateRequest | None = None


class UserUpdateRequest(BaseModel):
    username: Annotated[
        str | None,
        MinLen(3),
        MaxLen(20),
        AfterValidator(validate_username),
    ] = Field(
        default=None,
        description="Username must be between 3 and 20 characters, only letters, numbers, underscores",
    )
    email: Annotated[EmailStr | None, MaxLen(100)] = Field(
        default=None,
        description="Valid email address, max 100 chars",
    )
    password: Annotated[str | None, MinLen(8)] = Field(
        default=None,
        description="Password must contain at least 8 characters",
    )
    profile: ProfileCreateRequest | None = None


class UserResponse(BaseModel):
    id: int
    username: str
    email: EmailStr
    is_active: bool = True
    created_at: datetime
