import os

import uvicorn

from backend.create_app import create_app
from backend.presentation.router import router as http_router


main_app = create_app()

main_app.include_router(http_router)

if __name__ == "__main__":
    os.environ.setdefault("WATCHFILES_IGNORE_PERMISSION_DENIED", "true")
    uvicorn.run(
        "main:main_app",
        reload=True,
        host=os.getenv("APP_HOST", "127.0.0.1"),
        port=int(os.getenv("APP_PORT", "8000")),
    )
