from dishka.integrations.fastapi import DishkaRoute, FromDishka
from fastapi import APIRouter

from backend.application.use_cases.blocks import BlockUserUseCase, UnblockUserUseCase
from backend.domain.user.entity import User
from backend.presentation.shared.http.schemas import MessageResponse


router = APIRouter(prefix="/blocks", tags=["Blocks"], route_class=DishkaRoute)


@router.post("/{user_id:int}")
async def block_user(
    user_id: int,
    use_case: FromDishka[BlockUserUseCase],
    current_user: FromDishka[User],
) -> MessageResponse:
    result = await use_case.execute(current_user, user_id)
    return MessageResponse(message=result.message)


@router.delete("/{user_id:int}")
async def unblock_user(
    user_id: int,
    use_case: FromDishka[UnblockUserUseCase],
    current_user: FromDishka[User],
) -> MessageResponse:
    result = await use_case.execute(current_user, user_id)
    return MessageResponse(message=result.message)
