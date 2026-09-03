from typing import Annotated, Literal

from dishka.integrations.fastapi import DishkaRoute, FromDishka
from fastapi import APIRouter, Depends, File, Form, Query, UploadFile

from backend.application.commands import ProfileUpdateCommand
from backend.application.use_cases.music import ListMusicUseCase
from backend.application.use_cases.profiles import (
    CreatePhotoAlbumUseCase,
    DeleteProfilePhotoUseCase,
    GetAvatarHistoryUseCase,
    GetProfilePhotosUseCase,
    GetProfileUseCase,
    RemoveAvatarUseCase,
    SelectAvatarUseCase,
    UpdateProfileUseCase,
    UploadAvatarUseCase,
    UploadProfilePhotoUseCase,
)
from backend.application.use_cases.videos import ListVideoAlbumsUseCase
from backend.domain.user.entity import User
from backend.presentation.profiles.http.schemas import (
    AvatarSelectRequest,
    PhotoAlbumCreateRequest,
    ProfileUpdateRequest,
)
from backend.presentation.shared.http.auth import get_optional_current_user_from_cookie
from backend.presentation.shared.http.schemas import (
    AvatarHistoryResponse,
    AvatarRemoveResponse,
    AvatarUploadResponse,
    MessageResponse,
    PhotoAlbumEnvelopeResponse,
    ProfilePageResponse,
    ProfilePhotosResponse,
    UserEnvelopeResponse,
)
from backend.presentation.shared.http.serializers import (
    avatar_history_result_to_response,
    avatar_remove_result_to_response,
    avatar_upload_result_to_response,
    message_result_to_response,
    photo_album_result_to_response,
    profile_photos_result_to_response,
    profile_result_to_response,
    user_result_to_response,
)


router = APIRouter(prefix="/profile", tags=["Profile"], route_class=DishkaRoute)


@router.get("/{profile_id:int}")
async def get_profile(
    profile_id: int,
    use_case: FromDishka[GetProfileUseCase],
    current_user: FromDishka[User],
) -> ProfilePageResponse:
    result = await use_case.execute(current_user=current_user, profile_id=profile_id)
    return profile_result_to_response(result)


@router.patch("/{profile_id:int}")
async def update_profile(
    profile_id: int,
    body: ProfileUpdateRequest,
    use_case: FromDishka[UpdateProfileUseCase],
    current_user: FromDishka[User],
) -> UserEnvelopeResponse:
    result = await use_case.execute(
        current_user=current_user,
        profile_id=profile_id,
        command=ProfileUpdateCommand(**body.model_dump()),
    )
    return user_result_to_response(result)


@router.get("/{profile_id:int}/photos")
async def get_profile_photos(
    profile_id: int,
    use_case: FromDishka[GetProfilePhotosUseCase],
    current_user: FromDishka[User],
) -> ProfilePhotosResponse:
    result = await use_case.execute(current_user=current_user, profile_id=profile_id)
    return profile_photos_result_to_response(result)


@router.get("/{profile_id:int}/videos")
async def get_profile_videos(
    profile_id: int,
    use_case: FromDishka[ListVideoAlbumsUseCase],
    current_user: Annotated[User | None, Depends(get_optional_current_user_from_cookie)],
    tab: Literal["uploaded", "favorite", "viewed", "bookmarked"] = Query("uploaded"),
    q: str = Query("", max_length=200),
) -> dict[str, object]:
    viewer_id = current_user.require_id() if current_user else None
    owner_name = await use_case.get_owner_name(user_id=profile_id)
    if tab != "uploaded" and viewer_id != profile_id:
        return {"owner_name": owner_name, "albums": []}
    albums = await use_case.execute_for_user(user_id=profile_id, viewer_id=viewer_id, tab=tab, search=q)
    return {
        "owner_name": owner_name,
        "albums": [
            {
                "id": album.id,
                "title": album.title,
                "videos": [
                    {
                        "video_id": video.id,
                        "media_id": video.media_id,
                        "title": video.title,
                        "owner_id": video.owner_id,
                        "owner_name": video.owner_name,
                        "status": video.status,
                        "error": video.error_message,
                        "original_filename": video.original_filename,
                        "created_at": video.created_at,
                        "duration": video.duration,
                        "views_count": video.views_count,
                        "likes_count": video.likes_count,
                        "is_liked_by_current": video.is_liked_by_current,
                        "is_bookmarked_by_current": video.is_bookmarked_by_current,
                        "is_favorited_by_current": video.is_favorited_by_current,
                    }
                    for video in album.videos
                ],
            }
            for album in albums
        ],
    }


@router.get("/{profile_id:int}/music")
async def get_profile_music(
    profile_id: int,
    use_case: FromDishka[ListMusicUseCase],
    profile_use_case: FromDishka[GetProfileUseCase],
    current_user: FromDishka[User],
) -> dict[str, list[dict[str, object]]]:
    await profile_use_case.execute(current_user=current_user, profile_id=profile_id)
    tracks = await use_case.execute(user_id=profile_id)
    return {
        "tracks": [
            {
                "id": track.id,
                "media_id": track.media_id,
                "user_id": track.user_id,
                "title": track.title,
                "artist": track.artist,
                "duration": track.duration,
                "created_at": track.created_at,
            }
            for track in tracks
        ]
    }


@router.post("/photos/albums")
async def create_photo_album(
    body: PhotoAlbumCreateRequest,
    use_case: FromDishka[CreatePhotoAlbumUseCase],
    current_user: FromDishka[User],
) -> PhotoAlbumEnvelopeResponse:
    result = await use_case.execute(current_user=current_user, title=body.title)
    return photo_album_result_to_response(result)


@router.post("/photos/albums/{album_id:int}")
async def upload_profile_photo(
    album_id: int,
    use_case: FromDishka[UploadProfilePhotoUseCase],
    current_user: FromDishka[User],
    photo: UploadFile = File(...),
    caption: Annotated[str | None, Form()] = None,
) -> PhotoAlbumEnvelopeResponse:
    result = await use_case.execute(
        current_user=current_user,
        album_id=album_id,
        photo=photo,
        caption=caption,
    )
    return photo_album_result_to_response(result)


@router.delete("/photos/{photo_id:int}")
async def delete_profile_photo(
    photo_id: int,
    use_case: FromDishka[DeleteProfilePhotoUseCase],
    current_user: FromDishka[User],
) -> MessageResponse:
    result = await use_case.execute(current_user=current_user, photo_id=photo_id)
    return message_result_to_response(result)


@router.patch("/avatar")
async def upload_avatar(
    use_case: FromDishka[UploadAvatarUseCase],
    current_user: FromDishka[User],
    avatar: UploadFile = File(...),
) -> AvatarUploadResponse:
    result = await use_case.execute(current_user=current_user, avatar=avatar)
    return avatar_upload_result_to_response(result)


@router.get("/avatar/history")
async def get_avatar_history(
    use_case: FromDishka[GetAvatarHistoryUseCase],
    current_user: FromDishka[User],
) -> AvatarHistoryResponse:
    result = await use_case.execute(current_user=current_user)
    return avatar_history_result_to_response(result)


@router.patch("/avatar/select")
async def select_avatar(
    body: AvatarSelectRequest,
    use_case: FromDishka[SelectAvatarUseCase],
    current_user: FromDishka[User],
) -> AvatarUploadResponse:
    result = await use_case.execute(current_user=current_user, avatar_url=body.avatar_url)
    return avatar_upload_result_to_response(result)


@router.delete("/avatar")
async def remove_avatar(
    use_case: FromDishka[RemoveAvatarUseCase],
    current_user: FromDishka[User],
) -> AvatarRemoveResponse:
    result = await use_case.execute(current_user=current_user)
    return avatar_remove_result_to_response(result)
