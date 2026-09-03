from dishka.integrations.fastapi import DishkaRoute, FromDishka
from fastapi import APIRouter, Query

from backend.application.ports.search_repository import SearchResultView
from backend.application.use_cases.search import SearchUseCase


router = APIRouter(prefix="/search", tags=["Search"], route_class=DishkaRoute)


@router.get("")
async def search(
    use_case: FromDishka[SearchUseCase],
    q: str = Query("", max_length=200),
) -> dict[str, object]:
    return _result_to_response(await use_case.execute(query=q))


def _result_to_response(result: SearchResultView) -> dict[str, object]:
    return {
        "users": [
            {"id": user.id, "username": user.username, "full_name": user.full_name, "avatar": user.avatar}
            for user in result.users
        ],
        "posts": [
            {
                "id": post.id,
                "content": post.content,
                "author_id": post.author_id,
                "author_name": post.author_name,
                "created_at": post.created_at,
            }
            for post in result.posts
        ],
        "videos": [
            {
                "video_id": video.video_id,
                "media_id": video.media_id,
                "title": video.title,
                "owner_id": video.owner_id,
                "owner_name": video.owner_name,
                "created_at": video.created_at,
            }
            for video in result.videos
        ],
    }
