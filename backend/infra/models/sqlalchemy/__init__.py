from .base import Base
from .comment import Comment
from .friendship import Friendship
from .like import LikePost
from .outbox_event import OutboxEvent
from .post import Post
from .profile import Profile
from .profile_avatar import ProfileAvatar
from .profile_photo import ProfilePhoto
from .profile_photo_album import ProfilePhotoAlbum
from .subscription import Subscription
from .user import User


__all__ = (
    "Base",
    "Comment",
    "Friendship",
    "LikePost",
    "OutboxEvent",
    "Post",
    "Profile",
    "ProfileAvatar",
    "ProfilePhoto",
    "ProfilePhotoAlbum",
    "Subscription",
    "User",
)

# Указываем, что этот модуль содержит реализации инфраструктуры
__infra__ = True
