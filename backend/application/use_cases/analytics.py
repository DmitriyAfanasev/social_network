from backend.application.ports.analytics_repository import AnalyticsRepository
from backend.application.results import AnalyticsSummaryResult


class GetAnalyticsSummaryUseCase:
    def __init__(self, analytics_repository: AnalyticsRepository) -> None:
        self.analytics_repository = analytics_repository

    async def execute(self) -> AnalyticsSummaryResult:
        return await self.analytics_repository.get_summary()
