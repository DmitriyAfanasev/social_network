from fastapi import APIRouter

from backend.presentation.analytics.http import router as analytics_router
from backend.presentation.admin.http import router as admin_router
from backend.presentation.auth.http import router as auth_router
from backend.presentation.blocks.http import router as blocks_router
from backend.presentation.comments.http import router as comments_router
from backend.presentation.friends.http import router as friends_router
from backend.presentation.likes.http import router as likes_router
from backend.presentation.media_router import router as media_router
from backend.presentation.messages.http import router as messages_router
from backend.presentation.messages.ws.router import router as messages_ws_router
from backend.presentation.notifications.http.router import router as notifications_router
from backend.presentation.posts.http import router as posts_router
from backend.presentation.profiles.http import router as profiles_router


router = APIRouter()
router.include_router(analytics_router)
router.include_router(admin_router)
router.include_router(auth_router)
router.include_router(posts_router)
router.include_router(profiles_router)
router.include_router(comments_router)
router.include_router(likes_router)
router.include_router(messages_router)
router.include_router(messages_ws_router)
router.include_router(media_router)
router.include_router(friends_router)
router.include_router(blocks_router)
router.include_router(notifications_router)
