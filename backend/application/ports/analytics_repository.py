from abc import ABC, abstractmethod

from backend.application.results import AnalyticsSummaryResult


class AnalyticsRepository(ABC):
    @abstractmethod
    async def get_summary(self) -> AnalyticsSummaryResult:
        pass
