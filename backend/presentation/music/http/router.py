from typing import Annotated

from dishka.integrations.fastapi import DishkaRoute, FromDishka
from fastapi import APIRouter, File, Form, HTTPException, UploadFile, status

from backend.application.ports.music_repository import MusicTrackView
from backend.application.use_cases.music import DeleteMusicUseCase, ListMusicUseCase, UploadMusicUseCase
from backend.domain.user.entity import User


router = APIRouter(prefix="/music", tags=["Music"], route_class=DishkaRoute)


@router.get("")
async def list_my_music(
    use_case: FromDishka[ListMusicUseCase],
    user: FromDishka[User],
) -> dict[str, list[dict[str, object]]]:
    tracks = await use_case.execute(user_id=user.require_id())
    return {"tracks": [_track_to_response(track) for track in tracks]}


@router.post("", status_code=status.HTTP_201_CREATED)
async def upload_music(
    use_case: FromDishka[UploadMusicUseCase],
    user: FromDishka[User],
    file: Annotated[UploadFile, File(description="Аудиофайл")],
    title: Annotated[str | None, Form()] = None,
    artist: Annotated[str | None, Form()] = None,
) -> dict[str, object]:
    try:
        track = await use_case.execute(user=user, file=file, title=title, artist=artist)
    except ValueError as error:
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail=str(error)) from error
    return {"track": _track_to_response(track)}


@router.delete("/{track_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_music(
    track_id: int,
    use_case: FromDishka[DeleteMusicUseCase],
    user: FromDishka[User],
) -> None:
    if not await use_case.execute(user=user, track_id=track_id):
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="Трек не найден")


def _track_to_response(track: MusicTrackView) -> dict[str, object]:
    return {
        "id": track.id,
        "media_id": track.media_id,
        "user_id": track.user_id,
        "title": track.title,
        "artist": track.artist,
        "duration": track.duration,
        "created_at": track.created_at,
    }
