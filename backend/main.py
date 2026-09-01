import os

import uvicorn

from backend.presentation.router import router as http_router
from backend.create_app import create_app


main_app = create_app()

main_app.include_router(http_router)

if __name__ == "__main__":
    os.environ.setdefault("WATCHFILES_IGNORE_PERMISSION_DENIED", "true")
    uvicorn.run(
        "main:main_app",
        reload=True,
    )
