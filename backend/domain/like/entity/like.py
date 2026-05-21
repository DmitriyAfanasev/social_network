from dataclasses import dataclass


@dataclass(frozen=True, kw_only=True)
class LikePost:
    user_id: int
    post_id: int

    def __post_init__(self) -> None:
        if self.user_id <= 0:
            raise ValueError("user_id must be positive.")
        if self.post_id <= 0:
            raise ValueError("post_id must be positive.")
