from typing import Annotated, Literal

from dishka.integrations.fastapi import DishkaRoute, FromDishka
from fastapi import APIRouter, Depends, File, Form, HTTPException, Query, UploadFile, status

from backend.application.ports.video_repository import VideoAlbumView, VideoView
from backend.application.use_cases.videos import (
    CreateVideoAlbumUseCase,
    DeleteVideoAlbumUseCase,
    DeleteVideoUseCase,
    GetVideoUseCase,
    ListVideoAlbumsUseCase,
    RecordVideoViewUseCase,
    ToggleVideoBookmarkUseCase,
    ToggleVideoFavoriteUseCase,
    ToggleVideoLikeUseCase,
    UploadVideoUseCase,
)
from backend.domain.user.entity import User
from backend.presentation.shared.http.auth import get_optional_current_user_from_cookie


router = APIRouter(prefix="/videos", tags=["Videos"], route_class=DishkaRoute)


@router.post("", status_code=status.HTTP_201_CREATED)
async def upload_video(
    use_case: FromDishka[UploadVideoUseCase],
    user: FromDishka[User],
    file: Annotated[UploadFile, File(description="Исходный видеофайл")],
    album_id: Annotated[int | None, Form()] = None,
    title: Annotated[str | None, Form()] = None,
) -> dict[str, int | str]:
    result = await use_case.execute(user=user, file=file, album_id=album_id, title=title)
    return {"video_id": result.video_id, "media_id": result.media_id, "status": result.status}


@router.get("")
async def list_video_albums(
    use_case: FromDishka[ListVideoAlbumsUseCase],
    user: FromDishka[User],
    tab: Literal["uploaded", "favorite", "viewed", "bookmarked"] = Query("uploaded"),
    q: str = Query("", max_length=200),
) -> dict[str, object]:
    albums = await use_case.execute(user=user, tab=tab, search=q)
    return {"albums": [_album_to_response(a) for a in albums]}


@router.post("/albums", status_code=status.HTTP_201_CREATED)
async def create_video_album(
    use_case: FromDishka[CreateVideoAlbumUseCase], user: FromDishka[User], title: Annotated[str, Form()]
) -> dict[str, int | str]:
    return {"album_id": await use_case.execute(user=user, title=title), "title": title.strip()}


@router.delete("/albums/{album_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_video_album(album_id: int, use_case: FromDishka[DeleteVideoAlbumUseCase], user: FromDishka[User]) -> None:
    if not await use_case.execute(user=user, album_id=album_id):
        raise HTTPException(status_code=404, detail="Видеоальбом не найден")


@router.delete("/{video_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_video(video_id: int, use_case: FromDishka[DeleteVideoUseCase], user: FromDishka[User]) -> None:
    if not await use_case.execute(user=user, video_id=video_id):
        raise HTTPException(status_code=404, detail="Видео не найдено")


@router.post("/{video_id}/view")
async def record_video_view(
    video_id: int,
    use_case: FromDishka[RecordVideoViewUseCase],
    current_user: Annotated[User | None, Depends(get_optional_current_user_from_cookie)],
) -> dict[str, int]:
    views_count = await use_case.execute(
        video_id=video_id,
        user_id=current_user.require_id() if current_user else None,
    )
    if views_count is None:
        raise HTTPException(status_code=404, detail="Видео не найдено")
    return {"views_count": views_count}


@router.post("/{video_id}/like")
async def toggle_video_like(
    video_id: int,
    use_case: FromDishka[ToggleVideoLikeUseCase],
    user: FromDishka[User],
) -> dict[str, int | bool]:
    result = await use_case.execute(video_id=video_id, user=user)
    if result is None:
        raise HTTPException(status_code=404, detail="Видео не найдено")
    likes_count, liked = result
    return {"likes_count": likes_count, "liked": liked}


@router.post("/{video_id}/bookmark")
async def add_video_bookmark(
    video_id: int,
    use_case: FromDishka[ToggleVideoBookmarkUseCase],
    user: FromDishka[User],
) -> dict[str, bool]:
    bookmarked = await use_case.add(video_id=video_id, user=user)
    if bookmarked is None:
        raise HTTPException(status_code=404, detail="Видео не найдено")
    return {"bookmarked": bookmarked}


@router.delete("/{video_id}/bookmark")
async def remove_video_bookmark(
    video_id: int,
    use_case: FromDishka[ToggleVideoBookmarkUseCase],
    user: FromDishka[User],
) -> dict[str, bool]:
    bookmarked = await use_case.remove(video_id=video_id, user=user)
    if bookmarked is None:
        raise HTTPException(status_code=404, detail="Видео не найдено")
    return {"bookmarked": bookmarked}


@router.post("/{video_id}/favorite")
async def add_video_favorite(
    video_id: int,
    use_case: FromDishka[ToggleVideoFavoriteUseCase],
    user: FromDishka[User],
) -> dict[str, bool]:
    favorited = await use_case.add(video_id=video_id, user=user)
    if favorited is None:
        raise HTTPException(status_code=404, detail="Видео не найдено")
    return {"favorited": favorited}


@router.delete("/{video_id}/favorite")
async def remove_video_favorite(
    video_id: int,
    use_case: FromDishka[ToggleVideoFavoriteUseCase],
    user: FromDishka[User],
) -> dict[str, bool]:
    favorited = await use_case.remove(video_id=video_id, user=user)
    if favorited is None:
        raise HTTPException(status_code=404, detail="Видео не найдено")
    return {"favorited": favorited}


@router.get("/{video_id}")
async def get_video(
    video_id: int,
    use_case: FromDishka[GetVideoUseCase],
    current_user: Annotated[User | None, Depends(get_optional_current_user_from_cookie)],
) -> dict[str, object]:
    result = await use_case.execute(
        video_id=video_id,
        user_id=current_user.require_id() if current_user else None,
    )
    if result is None:
        from fastapi import HTTPException

        raise HTTPException(status_code=404, detail="Видео не найдено")
    return {
        "video_id": result.id,
        "media_id": result.media_id,
        "title": result.title,
        "owner_id": result.owner_id,
        "owner_name": result.owner_name,
        "status": result.status,
        "error": result.error_message,
        "original_filename": result.original_filename,
        "created_at": result.created_at,
        "duration": result.duration,
        "views_count": result.views_count,
        "likes_count": result.likes_count,
        "is_liked_by_current": result.is_liked_by_current,
        "is_bookmarked_by_current": result.is_bookmarked_by_current,
        "is_favorited_by_current": result.is_favorited_by_current,
        "renditions": [
            {"height": r.height, "object_key": r.object_key, "content_type": r.content_type, "size": r.size}
            for r in result.renditions
        ],
    }


def _video_to_response(value: VideoView) -> dict[str, object]:
    return {
        "video_id": value.id,
        "media_id": value.media_id,
        "title": value.title,
        "owner_id": value.owner_id,
        "owner_name": value.owner_name,
        "status": value.status,
        "error": value.error_message,
        "original_filename": value.original_filename,
        "created_at": value.created_at,
        "duration": value.duration,
        "views_count": value.views_count,
        "likes_count": value.likes_count,
        "is_liked_by_current": value.is_liked_by_current,
        "is_bookmarked_by_current": value.is_bookmarked_by_current,
        "is_favorited_by_current": value.is_favorited_by_current,
    }


def _album_to_response(album: VideoAlbumView) -> dict[str, object]:
    return {"id": album.id, "title": album.title, "videos": [_video_to_response(video) for video in album.videos]}
