from dataclasses import dataclass
from enum import StrEnum
from uuid import uuid4


class CallType(StrEnum):
    AUDIO = "audio"
    VIDEO = "video"


class CallStatus(StrEnum):
    RINGING = "ringing"
    ACTIVE = "active"
    REJECTED = "rejected"
    ENDED = "ended"


@dataclass(frozen=True, slots=True)
class CallSession:
    call_id: str
    caller_id: int
    callee_id: int
    call_type: CallType
    status: CallStatus = CallStatus.RINGING

    @classmethod
    def start(cls, caller_id: int, callee_id: int, call_type: CallType) -> "CallSession":
        if caller_id == callee_id:
            raise ValueError("Нельзя позвонить самому себе")
        return cls(str(uuid4()), caller_id, callee_id, call_type)

    def includes(self, user_id: int) -> bool:
        return user_id in {self.caller_id, self.callee_id}

    def accept(self, user_id: int) -> "CallSession":
        self._ensure_callee(user_id)
        self._ensure_status(CallStatus.RINGING)
        return self._replace(status=CallStatus.ACTIVE)

    def reject(self, user_id: int) -> "CallSession":
        self._ensure_callee(user_id)
        self._ensure_status(CallStatus.RINGING)
        return self._replace(status=CallStatus.REJECTED)

    def end(self, user_id: int) -> "CallSession":
        if not self.includes(user_id):
            raise PermissionError("Пользователь не участвует в звонке")
        if self.status not in {CallStatus.RINGING, CallStatus.ACTIVE}:
            raise ValueError("Звонок уже завершён")
        return self._replace(status=CallStatus.ENDED)

    def _ensure_callee(self, user_id: int) -> None:
        if user_id != self.callee_id:
            raise PermissionError("Только получатель может изменить приглашение")

    def _ensure_status(self, expected: CallStatus) -> None:
        if self.status != expected:
            raise ValueError("Недопустимый переход состояния звонка")

    def _replace(self, *, status: CallStatus) -> "CallSession":
        return CallSession(self.call_id, self.caller_id, self.callee_id, self.call_type, status)
