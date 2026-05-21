from dataclasses import dataclass
from typing import Any


@dataclass(slots=True)
class ApplicationError(Exception):
    message: str
    status_code: int = 400
    code: str = "application_error"
    details: Any = None


class AuthenticationError(ApplicationError):
    def __init__(self, message: str = "Authentication required", details: Any = None):
        super().__init__(
            message=message,
            status_code=401,
            code="authentication_error",
            details=details,
        )


class PermissionDeniedError(ApplicationError):
    def __init__(self, message: str = "Permission denied", details: Any = None):
        super().__init__(
            message=message,
            status_code=403,
            code="permission_denied",
            details=details,
        )


class NotFoundError(ApplicationError):
    def __init__(self, message: str = "Resource not found", details: Any = None):
        super().__init__(
            message=message,
            status_code=404,
            code="not_found",
            details=details,
        )


class ValidationAppError(ApplicationError):
    def __init__(self, message: str = "Validation error", details: Any = None):
        super().__init__(
            message=message,
            status_code=400,
            code="validation_error",
            details=details,
        )


class ExternalServiceError(ApplicationError):
    def __init__(self, message: str = "External service error", details: Any = None):
        super().__init__(
            message=message,
            status_code=502,
            code="external_service_error",
            details=details,
        )
