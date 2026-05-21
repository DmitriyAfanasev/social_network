from dishka.integrations.fastapi import DishkaRoute, FromDishka
from fastapi import APIRouter, status

from backend.application.commands import CreateCommentCommand
from backend.application.use_cases.comments import CreateCommentUseCase, GetCommentsUseCase
from backend.domain.user.entity import User
from backend.presentation.comments.http.schemas import CommentCreateRequest
from backend.presentation.shared.http.schemas import CommentResponse, CommentsPageResponse
from backend.presentation.shared.http.serializers import (
    comment_result_to_response,
    comments_page_result_to_response,
)


router = APIRouter(tags=["Comments"], route_class=DishkaRoute)


@router.get("/posts/{post_id}/comments", response_model=CommentsPageResponse)
async def get_comments(
    post_id: int,
    use_case: FromDishka[GetCommentsUseCase],
    offset: int = 0,
    limit: int = 5,
) -> CommentsPageResponse:
    result = await use_case.execute(post_id=post_id, offset=offset, limit=limit)
    return comments_page_result_to_response(result)


@router.post(
    "/posts/{post_id}/comments",
    response_model=CommentResponse,
    status_code=status.HTTP_201_CREATED,
)
async def create_comment(
    post_id: int,
    body: CommentCreateRequest,
    use_case: FromDishka[CreateCommentUseCase],
    current_user: FromDishka[User],
) -> CommentResponse:
    result = await use_case.execute(
        current_user=current_user,
        post_id=post_id,
        command=CreateCommentCommand(**body.model_dump()),
    )
    return comment_result_to_response(result)
