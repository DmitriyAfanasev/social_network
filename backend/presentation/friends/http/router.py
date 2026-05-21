from dishka.integrations.fastapi import DishkaRoute, FromDishka
from fastapi import APIRouter

from backend.application.use_cases.friends import (
    AddFriendUseCase,
    CancelSubscriptionUseCase,
    GetFriendsUseCase,
    RemoveFriendUseCase,
)
from backend.domain.user.entity import User
from backend.presentation.shared.http.schemas import FriendActionResponse, FriendsResponse
from backend.presentation.shared.http.serializers import (
    friend_action_result_to_response,
    friends_result_to_response,
)


router = APIRouter(prefix="/friends", tags=["Friends"], route_class=DishkaRoute)


@router.get("", response_model=FriendsResponse)
async def get_friends(
    use_case: FromDishka[GetFriendsUseCase],
    current_user: FromDishka[User],
) -> FriendsResponse:
    result = await use_case.execute(current_user=current_user)
    return friends_result_to_response(result)


@router.post("/{friend_id:int}", response_model=FriendActionResponse)
async def add_friend(
    friend_id: int,
    use_case: FromDishka[AddFriendUseCase],
    current_user: FromDishka[User],
) -> FriendActionResponse:
    result = await use_case.execute(current_user=current_user, friend_id=friend_id)
    return friend_action_result_to_response(result)


@router.delete("/{friend_id:int}", response_model=FriendActionResponse)
async def remove_friend(
    friend_id: int,
    use_case: FromDishka[RemoveFriendUseCase],
    current_user: FromDishka[User],
) -> FriendActionResponse:
    result = await use_case.execute(current_user=current_user, friend_id=friend_id)
    return friend_action_result_to_response(result)


@router.delete("/subscriptions/{target_id:int}", response_model=FriendActionResponse)
async def cancel_subscription(
    target_id: int,
    use_case: FromDishka[CancelSubscriptionUseCase],
    current_user: FromDishka[User],
) -> FriendActionResponse:
    result = await use_case.execute(current_user=current_user, target_id=target_id)
    return friend_action_result_to_response(result)
