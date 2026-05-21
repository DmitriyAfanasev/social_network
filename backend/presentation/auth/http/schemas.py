from pydantic import BaseModel, EmailStr, Field


class LoginRequest(BaseModel):
    email: EmailStr
    password: str


class RegistrationConfirmationRequest(BaseModel):
    email: EmailStr


class RegistrationConfirmationConfirmRequest(BaseModel):
    token: str


class RegisterUserRequest(BaseModel):
    username: str = Field(min_length=3, max_length=20)
    email: EmailStr
    password: str = Field(min_length=8)
    password2: str = Field(min_length=8)


class PasswordResetRequest(BaseModel):
    email: EmailStr


class PasswordResetConfirmRequest(BaseModel):
    token: str
    new_password: str = Field(min_length=8)
    confirm_password: str = Field(min_length=8)
