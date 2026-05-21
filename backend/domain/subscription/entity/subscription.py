from dataclasses import dataclass

from backend.domain.shared.base_entity import BaseEntity


@dataclass(kw_only=True)
class Subscription(BaseEntity):
    subscriber_id: int
    target_id: int

    def is_self_subscription(self) -> bool:
        return self.subscriber_id == self.target_id
