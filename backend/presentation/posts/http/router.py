from typing import Annotated

from dishka.integrations.fastapi import DishkaRoute, FromDishka
from fastapi import APIRouter, Depends, File, Form, UploadFile, status

from backend.application.commands import CreatePostCommand, UpdatePostCommand
from backend.application.use_cases.posts import (
    CreatePostUseCase,
    DeletePostUseCase,
    GetFeedUseCase,
    RemovePostImageUseCase,
    UpdatePostUseCase,
)
from backend.domain.user.entity import User
from backend.presentation.shared.http.auth import (
    get_optional_current_user_from_cookie,
)
from backend.presentation.shared.http.schemas import (
    FeedResponse,
    MessageResponse,
    PostEnvelopeResponse,
    RemovePostImageResponse,
)
from backend.presentation.shared.http.serializers import (
    feed_result_to_response,
    message_result_to_response,
    post_result_to_response,
    remove_post_image_result_to_response,
)


router = APIRouter(tags=["Posts"], route_class=DishkaRoute)


@router.get("/posts", response_model=FeedResponse)
async def get_posts(
    use_case: FromDishka[GetFeedUseCase],
    current_user: Annotated[User | None, Depends(get_optional_current_user_from_cookie)],
    page: int = 1,
) -> FeedResponse:
    result = await use_case.execute(current_user=current_user, page=page)
    return feed_result_to_response(result)


@router.post(
    "/posts",
    response_model=PostEnvelopeResponse,
    status_code=status.HTTP_201_CREATED,
)
async def create_post(
    use_case: FromDishka[CreatePostUseCase],
    current_user: FromDishka[User],
    content: Annotated[str | None, Form()] = None,
    image: UploadFile | None = File(None, description="Изображение поста"),
) -> PostEnvelopeResponse:
    result = await use_case.execute(
        current_user=current_user,
        command=CreatePostCommand(content=content, image=image),
    )
    return post_result_to_response(result)


@router.patch("/posts/{post_id}", response_model=PostEnvelopeResponse)
async def update_post(
    post_id: int,
    use_case: FromDishka[UpdatePostUseCase],
    current_user: FromDishka[User],
    content: Annotated[str | None, Form()] = None,
    image: UploadFile | None = File(None),
) -> PostEnvelopeResponse:
    result = await use_case.execute(
        current_user=current_user,
        post_id=post_id,
        command=UpdatePostCommand(content=content, image=image),
    )
    return post_result_to_response(result)


@router.delete("/posts/{post_id}", response_model=MessageResponse)
async def delete_post(
    post_id: int,
    use_case: FromDishka[DeletePostUseCase],
    current_user: FromDishka[User],
) -> MessageResponse:
    result = await use_case.execute(current_user=current_user, post_id=post_id)
    return message_result_to_response(result)


@router.delete("/posts/{post_id}/image", response_model=RemovePostImageResponse)
async def delete_post_image(
    post_id: int,
    use_case: FromDishka[RemovePostImageUseCase],
    current_user: FromDishka[User],
) -> RemovePostImageResponse:
    result = await use_case.execute(current_user=current_user, post_id=post_id)
    return remove_post_image_result_to_response(result)
