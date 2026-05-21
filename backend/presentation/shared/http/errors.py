from fastapi import Request
from fastapi.responses import JSONResponse

from backend.application.exceptions import ApplicationError


async def application_error_handler(
    request: Request,
    error: Exception,
) -> JSONResponse:
    if not isinstance(error, ApplicationError):
        raise error

    content = {
        "error": {
            "code": error.code,
            "message": error.message,
        }
    }
    if error.details is not None:
        content["error"]["details"] = error.details

    return JSONResponse(
        status_code=error.status_code,
        content=content,
    )
