from typing import Annotated

from dishka.integrations.fastapi import DishkaRoute, FromDishka
from fastapi import APIRouter, File, Form, UploadFile

from backend.application.commands import ProfileUpdateCommand
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
from backend.domain.user.entity import User
from backend.presentation.profiles.http.schemas import (
    AvatarSelectRequest,
    PhotoAlbumCreateRequest,
    ProfileUpdateRequest,
)
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


@router.get("/{profile_id:int}", response_model=ProfilePageResponse)
async def get_profile(
    profile_id: int,
    use_case: FromDishka[GetProfileUseCase],
    current_user: FromDishka[User],
) -> ProfilePageResponse:
    result = await use_case.execute(current_user=current_user, profile_id=profile_id)
    return profile_result_to_response(result)


@router.patch("/{profile_id:int}", response_model=UserEnvelopeResponse)
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


@router.get("/{profile_id:int}/photos", response_model=ProfilePhotosResponse)
async def get_profile_photos(
    profile_id: int,
    use_case: FromDishka[GetProfilePhotosUseCase],
    current_user: FromDishka[User],
) -> ProfilePhotosResponse:
    result = await use_case.execute(current_user=current_user, profile_id=profile_id)
    return profile_photos_result_to_response(result)


@router.post("/photos/albums", response_model=PhotoAlbumEnvelopeResponse)
async def create_photo_album(
    body: PhotoAlbumCreateRequest,
    use_case: FromDishka[CreatePhotoAlbumUseCase],
    current_user: FromDishka[User],
) -> PhotoAlbumEnvelopeResponse:
    result = await use_case.execute(current_user=current_user, title=body.title)
    return photo_album_result_to_response(result)


@router.post("/photos/albums/{album_id:int}", response_model=PhotoAlbumEnvelopeResponse)
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


@router.delete("/photos/{photo_id:int}", response_model=MessageResponse)
async def delete_profile_photo(
    photo_id: int,
    use_case: FromDishka[DeleteProfilePhotoUseCase],
    current_user: FromDishka[User],
) -> MessageResponse:
    result = await use_case.execute(current_user=current_user, photo_id=photo_id)
    return message_result_to_response(result)


@router.patch("/avatar", response_model=AvatarUploadResponse)
async def upload_avatar(
    use_case: FromDishka[UploadAvatarUseCase],
    current_user: FromDishka[User],
    avatar: UploadFile = File(...),
) -> AvatarUploadResponse:
    result = await use_case.execute(current_user=current_user, avatar=avatar)
    return avatar_upload_result_to_response(result)


@router.get("/avatar/history", response_model=AvatarHistoryResponse)
async def get_avatar_history(
    use_case: FromDishka[GetAvatarHistoryUseCase],
    current_user: FromDishka[User],
) -> AvatarHistoryResponse:
    result = await use_case.execute(current_user=current_user)
    return avatar_history_result_to_response(result)


@router.patch("/avatar/select", response_model=AvatarUploadResponse)
async def select_avatar(
    body: AvatarSelectRequest,
    use_case: FromDishka[SelectAvatarUseCase],
    current_user: FromDishka[User],
) -> AvatarUploadResponse:
    result = await use_case.execute(current_user=current_user, avatar_url=body.avatar_url)
    return avatar_upload_result_to_response(result)


@router.delete("/avatar", response_model=AvatarRemoveResponse)
async def remove_avatar(
    use_case: FromDishka[RemoveAvatarUseCase],
    current_user: FromDishka[User],
) -> AvatarRemoveResponse:
    result = await use_case.execute(current_user=current_user)
    return avatar_remove_result_to_response(result)
