from dataclasses import dataclass

from backend.domain.call import CallSession


@dataclass(frozen=True)
class AuthTokensDTO:
    access_token: str
    refresh_token: str


@dataclass(frozen=True)
class RemovePostImageDTO:
    success: bool
    message: str


@dataclass(frozen=True)
class ToggleLikeDTO:
    success: bool
    action: str
    likes_count: int
    liked: bool


@dataclass(frozen=True, slots=True)
class CallDTO:
    call_id: str
    caller_id: int
    callee_id: int
    call_type: str
    status: str

    @classmethod
    def from_domain(cls, session: CallSession) -> "CallDTO":
        return cls(
            call_id=session.call_id,
            caller_id=session.caller_id,
            callee_id=session.callee_id,
            call_type=session.call_type.value,
            status=session.status.value,
        )
