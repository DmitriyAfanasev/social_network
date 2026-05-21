from dishka.integrations.fastapi import DishkaRoute, FromDishka
from fastapi import APIRouter

from backend.application.results import AnalyticsSummaryResult
from backend.application.use_cases.analytics import GetAnalyticsSummaryUseCase
from backend.domain.user.entity import User
from backend.presentation.analytics.http.schemas import AnalyticsSummaryResponse


router = APIRouter(prefix="/analytics", tags=["Analytics"], route_class=DishkaRoute)


@router.get("/summary", response_model=AnalyticsSummaryResponse)
async def get_analytics_summary(
    use_case: FromDishka[GetAnalyticsSummaryUseCase],
    current_user: FromDishka[User],
) -> AnalyticsSummaryResponse:
    result = await use_case.execute()
    return analytics_summary_to_response(result)


def analytics_summary_to_response(result: AnalyticsSummaryResult) -> AnalyticsSummaryResponse:
    return AnalyticsSummaryResponse(
        events_count=result.events_count,
        unique_users=result.unique_users,
        users_registered=result.users_registered,
        posts_created=result.posts_created,
        comments_created=result.comments_created,
        likes_added=result.likes_added,
        likes_removed=result.likes_removed,
        photos_deleted=result.photos_deleted,
    )
