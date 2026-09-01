from dishka.integrations.fastapi import DishkaRoute, FromDishka
from fastapi import APIRouter

from backend.application.use_cases.friends import (
    AddFriendUseCase,
    CancelSubscriptionUseCase,
    GetFriendsUseCase,
    RemoveFriendUseCase,
)
from backend.domain.user.entity import User
from backend.presentation.messages.auth import require_user_id
from backend.presentation.messages.ws.ports import MessageConnectionManagerPort
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
    manager: FromDishka[MessageConnectionManagerPort],
) -> FriendActionResponse:
    result = await use_case.execute(current_user=current_user, friend_id=friend_id)
    event_type = "friend.accepted" if result.is_friend else "friend.requested"
    await manager.publish_notification(
        [friend_id],
        {
            "type": event_type,
            "actor_id": require_user_id(current_user),
            "message": (
                "Ваша заявка в друзья принята"
                if result.is_friend
                else "Вам отправили заявку в друзья"
            ),
        },
    )
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
