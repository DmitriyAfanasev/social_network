from dishka.integrations.fastapi import DishkaRoute, FromDishka
from fastapi import APIRouter, status

from backend.application.use_cases.likes import TogglePostLikeUseCase
from backend.domain.user.entity import User
from backend.presentation.shared.http.schemas import ToggleLikeResponse
from backend.presentation.shared.http.serializers import toggle_like_result_to_response


router = APIRouter(tags=["Likes"], route_class=DishkaRoute)


@router.post(
    "/posts/{post_id}/likes",
    response_model=ToggleLikeResponse,
    status_code=status.HTTP_200_OK,
)
async def toggle_post_like(
    post_id: int,
    use_case: FromDishka[TogglePostLikeUseCase],
    current_user: FromDishka[User],
) -> ToggleLikeResponse:
    result = await use_case.execute(current_user=current_user, post_id=post_id)
    return toggle_like_result_to_response(result)
