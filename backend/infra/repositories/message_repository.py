from datetime import UTC, datetime

from sqlalchemy import select
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
        conversation = await self.session.scalar(select(Conversation).where(Conversation.direct_key == direct_key))
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

    async def get_participant_ids(self, conversation_id: int) -> list[int]:
        result = await self.session.scalars(
            select(ConversationParticipant.user_id).where(
                ConversationParticipant.conversation_id == conversation_id,
            )
        )
        return list(result.all())

    async def _is_participant(self, conversation_id: int, user_id: int) -> bool:
        return (
            await self.session.scalar(
                select(ConversationParticipant.id).where(
                    ConversationParticipant.conversation_id == conversation_id,
                    ConversationParticipant.user_id == user_id,
                )
            )
            is not None
        )

    async def list_messages(
        self,
        conversation_id: int,
        user_id: int,
        offset: int,
        limit: int,
    ) -> tuple[list[Message], bool]:
        if not await self._is_participant(conversation_id, user_id):
            raise NotFoundError("Диалог не найден")
        conditions = [Message.conversation_id == conversation_id]
        statement = (
            select(Message)
            .where(*conditions)
            .options(selectinload(Message.sender))
            .order_by(Message.created_at.desc(), Message.id.desc())
        )

        # Use a server-side cursor and fetch rows in bounded batches. OFFSET
        # remains an API concern; it is applied while consuming the stream so
        # the database/client never materializes the complete conversation.
        stream = await self.session.stream_scalars(statement)
        messages: list[Message] = []
        skipped = 0
        try:
            async for message in stream.yield_per(50):
                if skipped < offset:
                    skipped += 1
                    continue
                messages.append(message)
                if len(messages) > limit:
                    break
        finally:
            await stream.close()

        has_more = len(messages) > limit
        return messages[:limit], has_more

    async def create_message(
        self, conversation_id: int, sender_id: int, text: str, media_id: int | None = None
    ) -> Message:
        if not await self._is_participant(conversation_id, sender_id):
            raise NotFoundError("Диалог не найден")
        message = Message(conversation_id=conversation_id, sender_id=sender_id, text=text, media_id=media_id)
        self.session.add(message)
        await self.session.flush()
        await self.session.refresh(message, attribute_names=["created_at"])
        return message

    async def update_message(self, conversation_id: int, message_id: int, sender_id: int, text: str) -> Message:
        """Обновляет текст авторского сообщения."""
        message = await self._owned_message(conversation_id, message_id, sender_id)
        if message.deleted_at is not None:
            raise NotFoundError("Сообщение не найдено")
        message.text = text
        message.edited_at = datetime.now(UTC)
        await self.session.flush()
        return message

    async def remove_message_media(self, conversation_id: int, message_id: int, sender_id: int) -> tuple[Message, int | None]:
        message = await self._owned_message(conversation_id, message_id, sender_id)
        if message.deleted_at is not None:
            raise NotFoundError("Сообщение не найдено")
        media_id = message.media_id
        message.media_id = None
        message.edited_at = datetime.now(UTC)
        await self.session.flush()
        return message, media_id

    async def delete_message(self, conversation_id: int, message_id: int, sender_id: int) -> Message:
        """Помечает авторское сообщение удалённым."""
        message = await self._owned_message(conversation_id, message_id, sender_id)
        message.text = ""
        message.deleted_at = datetime.now(UTC)
        await self.session.flush()
        return message

    async def _owned_message(self, conversation_id: int, message_id: int, sender_id: int) -> Message:
        """Возвращает сообщение автора или сообщает, что оно недоступно."""
        message = await self.session.scalar(
            select(Message).where(
                Message.id == message_id,
                Message.conversation_id == conversation_id,
                Message.sender_id == sender_id,
            )
        )
        if message is None:
            raise NotFoundError("Сообщение не найдено или недоступно")
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
            current = await self.session.scalar(select(Message).where(Message.id == participant.last_read_message_id))
        if current is None or (message.created_at, message.id) > (current.created_at, current.id):
            participant.last_read_message_id = message.id
            # The column is TIMESTAMP WITHOUT TIME ZONE, matching the
            # project's existing CreatedAtMixin representation.
            participant.read_at = datetime.now()
        await self.session.flush()
        return True
