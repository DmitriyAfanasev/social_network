from datetime import UTC, datetime
from uuid import uuid4

from backend.application.events import JsonPayload


def build_analytics_event_payload(
    *,
    user_id: int | None,
    entity_type: str,
    entity_id: int | None,
    data: JsonPayload | None = None,
) -> JsonPayload:
    return {
        "event_id": str(uuid4()),
        "occurred_at": datetime.now(UTC).isoformat(),
        "user_id": user_id,
        "entity_type": entity_type,
        "entity_id": entity_id,
        "data": data or {},
    }
