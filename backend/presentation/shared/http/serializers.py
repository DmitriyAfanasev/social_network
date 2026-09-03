from typing import Any

from backend.application.results import (
    AuthResult,
    AvatarHistoryResult,
    AvatarRemoveResult,
    AvatarUploadResult,
    CommentResult,
    CommentsPageResult,
    FeedResult,
    FriendActionResult,
    FriendRecommendationsResult,
    FriendsResult,
    MessageResult,
    PhotoAlbumResult,
    PostResult,
    ProfilePhotoAlbumResult,
    ProfilePhotosResult,
    ProfileResult,
    RemovePostImageResult,
    ToggleLikeResult,
    UserResult,
)
from backend.presentation.shared.http.schemas import (
    AuthResponse,
    AvatarHistoryItemResponse,
    AvatarHistoryResponse,
    AvatarRemoveResponse,
    AvatarUploadResponse,
    CommentResponse,
    CommentsPageResponse,
    FeedResponse,
    FriendActionResponse,
    FriendRecommendationResponse,
    FriendRecommendationsResponse,
    FriendsResponse,
    MessageResponse,
    PhotoAlbumEnvelopeResponse,
    PostEnvelopeResponse,
    PostResponse,
    ProfilePageResponse,
    ProfilePhotoAlbumResponse,
    ProfilePhotoResponse,
    ProfilePhotosResponse,
    ProfilePreviewResponse,
    ProfileResponse,
    RemovePostImageResponse,
    ToggleLikeResponse,
    UserEnvelopeResponse,
    UserResponse,
)


def profile_to_response(profile: Any | None, *, own_profile: bool = False) -> ProfileResponse | None:
    if profile is None:
        return None

    return ProfileResponse(
        first_name=profile.first_name,
        last_name=profile.last_name,
        middle_name=profile.middle_name,
        full_name=profile.full_name,
        birth_date=profile.birth_date if own_profile or profile.show_birth_date else None,
        gender=profile.gender,
        phone_number=profile.phone_number if own_profile or profile.show_phone else None,
        country=profile.country,
        city=profile.city,
        street=profile.street,
        bio=profile.bio,
        status=profile.status,
        avatar=profile.avatar,
    )


def user_to_response(
    user: Any | None,
    *,
    include_email: bool = False,
    own_profile: bool = False,
) -> UserResponse | None:
    if user is None:
        return None

    return UserResponse(
        id=user.id,
        username=user.username,
        email=user.email if include_email else None,
        is_active=user.is_active,
        is_superuser=user.is_superuser,
        created_at=user.created_at,
        last_seen_at=user.last_seen_at,
        profile=profile_to_response(user.profile, own_profile=own_profile),
    )


def required_user_to_response(user: Any) -> UserResponse:
    response = user_to_response(user)
    if response is None:
        raise ValueError("Рекомендация содержит пользователя без данных")
    return response


def post_to_response(post: Any) -> PostResponse:
    return PostResponse(
        id=post.id,
        content=post.content,
        image=post.image,
        image_content_type=getattr(post, "image_content_type", None),
        author_id=post.author_id,
        author=(
            user_to_response(post.author)
            if getattr(post, "author", None)
            else None
        ),
        likes_count=getattr(post, "likes_count", 0),
        comments_count=getattr(post, "count_comments", 0),
        is_liked_by_current=getattr(post, "is_liked_by_current", False),
        liked_user_ids=list(getattr(post, "liked_user_ids", set())),
        liked_users=[
            user_response
            for user in getattr(post, "liked_users", [])
            if (user_response := user_to_response(user)) is not None
        ],
        created_at=post.created_at,
        updated_at=post.updated_at,
        featured_comment=(
            comment_to_response(post.preview_comment)
            if getattr(post, "preview_comment", None)
            else None
        ),
    )


def comment_to_response(
    comment: Any,
    *,
    include_author: bool = True,
) -> CommentResponse:
    return CommentResponse(
        id=comment.id,
        post_id=comment.post_id,
        user_id=comment.user_id,
        parent_id=comment.parent_id,
        text=comment.text,
        author=(
            user_to_response(comment.user)
            if include_author and getattr(comment, "user", None)
            else None
        ),
        created_at=comment.created_at,
        updated_at=comment.updated_at,
        likes_count=getattr(comment, "likes_count", 0),
        is_liked_by_current=getattr(comment, "is_liked_by_current", False),
    )


def auth_result_to_response(result: AuthResult) -> AuthResponse:
    return AuthResponse(user=user_to_response(result.user, include_email=True))


def message_result_to_response(result: MessageResult) -> MessageResponse:
    return MessageResponse(message=result.message)


def feed_result_to_response(result: FeedResult) -> FeedResponse:
    return FeedResponse(
        current_user=user_to_response(result.current_user, include_email=True),
        posts=[post_to_response(post) for post in result.posts],
        page=result.page,
        total_pages=result.total_pages,
    )


def post_result_to_response(result: PostResult) -> PostEnvelopeResponse:
    post = post_to_response(result.post)
    if result.author is not None:
        post.author = user_to_response(result.author)
    return PostEnvelopeResponse(post=post)


def remove_post_image_result_to_response(
    result: RemovePostImageResult,
) -> RemovePostImageResponse:
    return RemovePostImageResponse(success=result.success, message=result.message)


def comment_result_to_response(result: CommentResult) -> CommentResponse:
    comment = comment_to_response(result.comment, include_author=result.author is None)
    if result.author is not None:
        comment.author = user_to_response(result.author)
    return comment


def comments_page_result_to_response(result: CommentsPageResult) -> CommentsPageResponse:
    return CommentsPageResponse(
        comments=[comment_to_response(comment) for comment in result.comments],
        has_more=result.has_more,
        offset=result.offset,
        limit=result.limit,
        post_id=result.post_id,
    )


def toggle_like_result_to_response(result: ToggleLikeResult) -> ToggleLikeResponse:
    return ToggleLikeResponse(
        success=result.success,
        action=result.action,
        likes_count=result.likes_count,
        liked=result.liked,
    )


def profile_result_to_response(result: ProfileResult) -> ProfilePageResponse:
    return ProfilePageResponse(
        user=user_to_response(
            result.user,
            include_email=result.is_own_profile
            or bool(result.user.profile and result.user.profile.show_email),
            own_profile=result.is_own_profile,
        ),
        is_own_profile=result.is_own_profile,
        is_friend=result.is_friend,
        is_subscribed=result.is_subscribed,
        is_subscribed_to_current=result.is_subscribed_to_current,
        can_send_friend_request=result.can_send_friend_request,
        can_send_message=result.can_send_message,
        relationship_status=_relationship_status(result),
        current_user=user_to_response(result.current_user, include_email=True),
        posts=[post_to_response(post) for post in result.posts],
        friends=[friend_response for friend in result.friends if (friend_response := user_to_response(friend)) is not None],
        profile_visibility=result.user.profile.profile_visibility if result.is_own_profile and result.user.profile else None,
        friend_request_policy=result.user.profile.friend_request_policy if result.is_own_profile and result.user.profile else None,
        message_policy=result.user.profile.message_policy if result.is_own_profile and result.user.profile else None,
        show_email=result.user.profile.show_email if result.is_own_profile and result.user.profile else None,
        show_phone=result.user.profile.show_phone if result.is_own_profile and result.user.profile else None,
        show_birth_date=result.user.profile.show_birth_date if result.is_own_profile and result.user.profile else None,
        show_friends=result.user.profile.show_friends if result.is_own_profile and result.user.profile else None,
        show_posts=result.user.profile.show_posts if result.is_own_profile and result.user.profile else None,
    )


def profile_preview_to_response(user: Any) -> ProfilePreviewResponse:
    profile = getattr(user, "profile", None)
    return ProfilePreviewResponse(
        id=user.id,
        username=user.username,
        full_name=user.full_name,
        avatar=profile.avatar if profile else None,
        last_seen_at=user.last_seen_at,
    )


def _relationship_status(result: ProfileResult) -> str:
    if result.is_own_profile:
        return "self"
    if result.is_friend:
        return "friend"
    if result.is_subscribed_to_current:
        return "incoming_request"
    if result.is_subscribed:
        return "outgoing_request"
    if not result.can_send_friend_request:
        return "profile_private"
    if not result.can_send_message:
        return "messages_restricted"
    return "not_friend"


def user_result_to_response(result: UserResult) -> UserEnvelopeResponse:
    return UserEnvelopeResponse(user=user_to_response(result.user, include_email=True))


def avatar_upload_result_to_response(
    result: AvatarUploadResult,
) -> AvatarUploadResponse:
    return AvatarUploadResponse(message=result.message, avatar_url=result.avatar_url)


def avatar_remove_result_to_response(
    result: AvatarRemoveResult,
) -> AvatarRemoveResponse:
    return AvatarRemoveResponse(new_avatar=result.new_avatar, message=result.message)


def avatar_history_result_to_response(result: AvatarHistoryResult) -> AvatarHistoryResponse:
    return AvatarHistoryResponse(
        current_avatar=result.current_avatar,
        avatars=[
            AvatarHistoryItemResponse(
                id=avatar.id,
                avatar_url=avatar.avatar_url,
                created_at=avatar.created_at,
                is_current=avatar.avatar_url == result.current_avatar,
            )
            for avatar in result.avatars
        ],
    )


def profile_photo_album_to_response(album: ProfilePhotoAlbumResult) -> ProfilePhotoAlbumResponse:
    return ProfilePhotoAlbumResponse(
        id=album.id,
        title=album.title,
        kind=album.kind,
        created_at=album.created_at,
        photos=[
            ProfilePhotoResponse(
                id=photo.id,
                album_id=photo.album_id,
                photo_url=photo.photo_url,
                caption=photo.caption,
                created_at=photo.created_at,
            )
            for photo in album.photos
        ],
    )


def profile_photos_result_to_response(result: ProfilePhotosResult) -> ProfilePhotosResponse:
    return ProfilePhotosResponse(
        user=user_to_response(result.user),
        is_own_profile=result.is_own_profile,
        albums=[profile_photo_album_to_response(album) for album in result.albums],
    )


def photo_album_result_to_response(result: PhotoAlbumResult) -> PhotoAlbumEnvelopeResponse:
    return PhotoAlbumEnvelopeResponse(album=profile_photo_album_to_response(result.album))


def friends_result_to_response(result: FriendsResult) -> FriendsResponse:
    return FriendsResponse(
        current_user=user_to_response(result.current_user, include_email=True),
        friends=[
            friend_response
            for friend in result.friends
            if (friend_response := user_to_response(friend)) is not None
        ],
        subscribers=[
            subscriber_response
            for subscriber in result.subscribers
            if (subscriber_response := user_to_response(subscriber)) is not None
        ],
        subscriptions=[
            subscription_response
            for subscription in result.subscriptions
            if (subscription_response := user_to_response(subscription)) is not None
        ],
    )


def friend_recommendations_result_to_response(
    result: FriendRecommendationsResult,
) -> FriendRecommendationsResponse:
    return FriendRecommendationsResponse(
        recommendations=[
            FriendRecommendationResponse(
                user=required_user_to_response(item.user),
                common_friends=item.common_friends,
            )
            for item in result.recommendations
        ]
    )


def friend_action_result_to_response(result: FriendActionResult) -> FriendActionResponse:
    return FriendActionResponse(
        success=result.success,
        is_friend=result.is_friend,
        is_subscribed=result.is_subscribed,
        is_subscribed_to_current=result.is_subscribed_to_current,
        message=result.message,
    )
