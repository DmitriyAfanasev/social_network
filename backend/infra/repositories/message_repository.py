from datetime import datetime

from sqlalchemy import and_, or_, select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from backend.application.exceptions import NotFoundError
from backend.application.ports.message_repository import MessageRepository as MessageRepositoryPort
from backend.infra.models.sqlalchemy import Conversation, ConversationParticipant, Message, User


class MessageRepository(MessageRepositoryPort):
    def __init__(self, session: AsyncSession) -> None:
        self.session = session

    async def get_or_create_direct(self, user_id: int, other_user_id: int) -> Conversation:
        left, right = sorted((user_id, other_user_id))
        direct_key = f"{left}:{right}"
        conversation = await self.session.scalar(
            select(Conversation).where(Conversation.direct_key == direct_key)
        )
        if conversation is not None:
            return conversation

        if await self.session.scalar(select(User.id).where(User.id == other_user_id)) is None:
            raise NotFoundError("Пользователь не найден")

        conversation = Conversation(direct_key=direct_key)
        conversation.participants = [
            ConversationParticipant(user_id=left),
            ConversationParticipant(user_id=right),
        ]
        self.session.add(conversation)
        await self.session.flush()
        return conversation

    async def list_conversations(self, user_id: int) -> list[Conversation]:
        result = await self.session.execute(
            select(Conversation)
            .join(ConversationParticipant)
            .where(ConversationParticipant.user_id == user_id)
            .options(selectinload(Conversation.participants).selectinload(ConversationParticipant.user))
            .order_by(Conversation.created_at.desc(), Conversation.id.desc())
        )
        return list(result.scalars().unique().all())

    async def _is_participant(self, conversation_id: int, user_id: int) -> bool:
        return await self.session.scalar(
            select(ConversationParticipant.id).where(
                ConversationParticipant.conversation_id == conversation_id,
                ConversationParticipant.user_id == user_id,
            )
        ) is not None

    async def list_messages(
        self,
        conversation_id: int,
        user_id: int,
        before: tuple[datetime, int] | None,
        limit: int,
    ) -> tuple[list[Message], bool]:
        if not await self._is_participant(conversation_id, user_id):
            raise NotFoundError("Диалог не найден")
        conditions = [Message.conversation_id == conversation_id]
        if before is not None:
            before_at, before_id = before
            conditions.append(
                or_(
                    Message.created_at < before_at,
                    and_(Message.created_at == before_at, Message.id < before_id),
                )
            )
        result = await self.session.execute(
            select(Message)
            .where(*conditions)
            .options(selectinload(Message.sender))
            .order_by(Message.created_at.desc(), Message.id.desc())
            .limit(limit)
        )
        messages = list(result.scalars().all())
        return messages, len(messages) == limit

    async def create_message(self, conversation_id: int, sender_id: int, text: str) -> Message:
        if not await self._is_participant(conversation_id, sender_id):
            raise NotFoundError("Диалог не найден")
        message = Message(conversation_id=conversation_id, sender_id=sender_id, text=text)
        self.session.add(message)
        await self.session.flush()
        await self.session.refresh(message, attribute_names=["created_at"])
        return message

    async def mark_read(self, conversation_id: int, user_id: int, message_id: int) -> bool:
        participant = await self.session.scalar(
            select(ConversationParticipant).where(
                ConversationParticipant.conversation_id == conversation_id,
                ConversationParticipant.user_id == user_id,
            )
        )
        message = await self.session.scalar(
            select(Message).where(
                Message.id == message_id,
                Message.conversation_id == conversation_id,
            )
        )
        if participant is None or message is None:
            return False
        current = None
        if participant.last_read_message_id is not None:
            current = await self.session.scalar(
                select(Message).where(Message.id == participant.last_read_message_id)
            )
        if current is None or (message.created_at, message.id) > (current.created_at, current.id):
            participant.last_read_message_id = message.id
            # The column is TIMESTAMP WITHOUT TIME ZONE, matching the
            # project's existing CreatedAtMixin representation.
            participant.read_at = datetime.now()
        await self.session.flush()
        return True
