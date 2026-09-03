from backend.application.ports.search_repository import SearchRepository, SearchResultView


class SearchUseCase:
    def __init__(self, repository: SearchRepository) -> None:
        self.repository = repository

    async def execute(self, *, query: str, limit: int = 20) -> SearchResultView:
        normalized_query = query.strip()
        if not normalized_query:
            return SearchResultView(users=(), posts=(), videos=())
        return await self.repository.search(query=normalized_query, limit=max(1, min(limit, 50)))
