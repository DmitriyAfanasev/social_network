from dishka.integrations.fastapi import DishkaRoute, FromDishka
from fastapi import APIRouter, HTTPException, Response, status

from backend.application.ports.media_storage import MediaStorage


router = APIRouter(tags=["Media"], route_class=DishkaRoute)


@router.get("/media/default-avatar")
async def get_default_avatar(media_storage: FromDishka[MediaStorage]) -> Response:
    """Возвращает системную дефолтную аватарку из MinIO."""
    result = await media_storage.get_default_avatar()
    if result is None:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="Дефолтный аватар не загружен")
    metadata, content = result
    return Response(content=content, media_type=metadata.content_type or "image/svg+xml")


@router.get("/media/{media_id}")
async def get_media(media_id: int, media_storage: FromDishka[MediaStorage]) -> Response:
    """Возвращает медиафайл из MinIO по его идентификатору."""
    result = await media_storage.get(media_id)
    if result is None:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="Медиафайл не найден")
    metadata, content = result
    return Response(content=content, media_type=metadata.content_type or "application/octet-stream")
