from backend.application.dto import CallDTO
from backend.application.exceptions import NotFoundError, PermissionDeniedError
from backend.application.ports.call_session_store import CallSessionStore
from backend.application.use_cases.messages import MessagingUseCase
from backend.domain.call import CallSession, CallType


class CallUseCase:
    """Manages call lifecycle and authorizes WebRTC signaling participants."""

    def __init__(self, store: CallSessionStore, messaging: MessagingUseCase) -> None:
        self.store = store
        self.messaging = messaging

    async def start(self, caller_id: int, callee_id: int, call_type: CallType) -> CallDTO:
        if not await self.messaging.can_send_message(caller_id, callee_id):
            raise PermissionDeniedError("Пользователь недоступен для звонка")
        session = CallSession.start(caller_id, callee_id, call_type)
        await self.store.save(session)
        return CallDTO.from_domain(session)

    async def accept(self, call_id: str, user_id: int) -> CallDTO:
        session = await self._get(call_id)
        try:
            updated = session.accept(user_id)
        except (PermissionError, ValueError) as error:
            raise PermissionDeniedError(str(error)) from error
        await self.store.save(updated)
        return CallDTO.from_domain(updated)

    async def reject(self, call_id: str, user_id: int) -> CallDTO:
        session = await self._get(call_id)
        try:
            updated = session.reject(user_id)
        except (PermissionError, ValueError) as error:
            raise PermissionDeniedError(str(error)) from error
        await self.store.save(updated)
        return CallDTO.from_domain(updated)

    async def end(self, call_id: str, user_id: int) -> CallDTO:
        session = await self._get(call_id)
        try:
            updated = session.end(user_id)
        except (PermissionError, ValueError) as error:
            raise PermissionDeniedError(str(error)) from error
        await self.store.save(updated)
        return CallDTO.from_domain(updated)

    async def authorize_signaling(self, call_id: str, user_id: int) -> CallDTO:
        session = await self._get(call_id)
        if not session.includes(user_id):
            raise PermissionDeniedError("Пользователь не участвует в звонке")
        return CallDTO.from_domain(session)

    async def _get(self, call_id: str) -> CallSession:
        session = await self.store.get(call_id)
        if session is None:
            raise NotFoundError("Звонок не найден")
        return session
