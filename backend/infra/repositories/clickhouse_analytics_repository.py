from backend.application.ports.analytics_repository import AnalyticsRepository as AnalyticsRepositoryPort
from backend.application.results import AnalyticsSummaryResult
from backend.infra.analytics.clickhouse_client import ClickHouseAnalyticsClient


class ClickHouseAnalyticsRepository(AnalyticsRepositoryPort):
    def __init__(self, client: ClickHouseAnalyticsClient) -> None:
        self.client = client

    async def get_summary(self) -> AnalyticsSummaryResult:
        return await self.client.get_summary()
