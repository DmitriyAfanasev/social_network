import pytest

from backend.domain.call import CallSession, CallStatus, CallType


def make_call() -> CallSession:
    return CallSession.start(1, 2, CallType.AUDIO)


def test_call_lifecycle_is_explicit_and_immutable() -> None:
    call = make_call()

    ringing = call
    active = ringing.accept(2)
    ended = active.end(1)

    assert ringing.status is CallStatus.RINGING
    assert active.status is CallStatus.ACTIVE
    assert ended.status is CallStatus.ENDED
    assert ended.call_id == call.call_id


@pytest.mark.parametrize("action", ["accept", "reject"])
def test_only_callee_can_answer_invitation(action: str) -> None:
    with pytest.raises(PermissionError):
        getattr(make_call(), action)(1)


def test_call_rejects_invalid_transitions_and_non_participants() -> None:
    rejected = make_call().reject(2)
    with pytest.raises(ValueError, match="уже завершён"):
        rejected.end(1)
    with pytest.raises(PermissionError):
        make_call().end(3)


def test_call_cannot_start_with_self() -> None:
    with pytest.raises(ValueError, match="самому себе"):
        CallSession.start(1, 1, CallType.VIDEO)
