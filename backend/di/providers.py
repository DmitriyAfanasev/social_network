"""Dishka providers for infrastructure and application services."""

from collections.abc import AsyncIterator

from dishka import Provider, Scope, provide
from sqlalchemy.ext.asyncio import (
    AsyncEngine,
    AsyncSession,
    async_sessionmaker,
    create_async_engine,
)

from backend.application.ports.admin_repository import AdminRepository
from backend.application.ports.analytics_repository import AnalyticsRepository
from backend.application.ports.audit_repository import AuditRepository
from backend.application.ports.authorization import AuthorizationService
from backend.application.ports.block_repository import BlockRepository
from backend.application.ports.comment_repository import CommentRepository
from backend.application.ports.file_upload_service import FileUploadService
from backend.application.ports.friend_repository import FriendRepository
from backend.application.ports.like_repository import LikeRepository
from backend.application.ports.media_storage import MediaStorage
from backend.application.ports.message_event_broker import MessageEventBroker
from backend.application.ports.message_repository import MessageRepository
from backend.application.ports.notification_sender import NotificationSender
from backend.application.ports.outbox_repository import OutboxRepository
from backend.application.ports.password_hasher import PasswordHasher
from backend.application.ports.post_repository import PostRepository as PostRepositoryPort
from backend.application.ports.profile_repository import ProfileRepository
from backend.application.ports.token_service import AuthTokenService
from backend.application.ports.token_store import PendingTokenStore
from backend.application.ports.transaction_manager import TransactionManager
from backend.application.ports.user_repository import UserRepository as UserRepositoryPort
from backend.application.use_cases.admin import AdminRbacUseCase
from backend.application.use_cases.analytics import GetAnalyticsSummaryUseCase
from backend.application.use_cases.auth import (
    ConfirmRegistrationUseCase,
    GetCurrentUserUseCase,
    LoginUseCase,
    RefreshTokenUseCase,
    RegisterUserUseCase,
    RequestPasswordResetUseCase,
    RequestRegistrationConfirmationUseCase,
    ResetPasswordUseCase,
)
from backend.application.use_cases.blocks import BlockUserUseCase, UnblockUserUseCase
from backend.application.use_cases.comment_likes import ToggleCommentLikeUseCase
from backend.application.use_cases.comments import (
    CreateCommentUseCase,
    DeleteCommentUseCase,
    GetCommentsUseCase,
    UpdateCommentUseCase,
)
from backend.application.use_cases.friends import (
    AddFriendUseCase,
    CancelSubscriptionUseCase,
    GetFriendRecommendationsUseCase,
    GetFriendsUseCase,
    RemoveFriendUseCase,
)
from backend.application.use_cases.likes import TogglePostLikeUseCase
from backend.application.use_cases.messages import MessagingUseCase
from backend.application.use_cases.posts import (
    CreatePostUseCase,
    DeletePostUseCase,
    GetFeedUseCase,
    RemovePostImageUseCase,
    UpdatePostUseCase,
)
from backend.application.use_cases.profiles import (
    CreatePhotoAlbumUseCase,
    DeleteProfilePhotoUseCase,
    GetAvatarHistoryUseCase,
    GetProfilePhotosUseCase,
    GetProfileUseCase,
    RemoveAvatarUseCase,
    SelectAvatarUseCase,
    UpdateProfileUseCase,
    UploadAvatarUseCase,
    UploadProfilePhotoUseCase,
)
from backend.application.use_cases.users import TouchUserActivityUseCase
from backend.infra.analytics.clickhouse_client import ClickHouseAnalyticsClient
from backend.infra.config import (
    ClickHouseConfig,
    DatabaseConfig,
    FileStorageConfig,
    FrontendConfig,
    JwtConfig,
    RedisConfig,
    Settings,
    settings,
)
from backend.infra.messaging.redis_message_event_broker import RedisMessageEventBroker
from backend.infra.notifications.outbox_notification_sender import OutboxNotificationSender
from backend.infra.repositories.admin_repository import AdminRepository as InfraAdminRepository
from backend.infra.repositories.audit_repository import AuditRepository as InfraAuditRepository
from backend.infra.repositories.block_repository import BlockRepository as InfraBlockRepository
from backend.infra.repositories.clickhouse_analytics_repository import ClickHouseAnalyticsRepository
from backend.infra.repositories.comment_repository import CommentRepository as InfraCommentRepository
from backend.infra.repositories.friend_repository import FriendRepository as InfraFriendRepository
from backend.infra.repositories.like_repository import LikeRepository as InfraLikeRepository
from backend.infra.repositories.message_repository import MessageRepository as InfraMessageRepository
from backend.infra.repositories.outbox_repository import OutboxRepository as InfraOutboxRepository
from backend.infra.repositories.pending_token_store import RedisPendingTokenStore
from backend.infra.repositories.post_repository import PostRepository as InfraPostRepository
from backend.infra.repositories.profile_repository import ProfileRepository as InfraProfileRepository
from backend.infra.repositories.user_repository import UserRepository as InfraUserRepository
from backend.infra.security.authorization import AuthorizationService as InfraAuthorizationService
from backend.infra.security.jwt_token_service import JwtAuthTokenService
from backend.infra.security.password_hasher import BcryptPasswordHasher
from backend.infra.storage.factory import create_file_upload_service
from backend.infra.storage.s3_media_storage import S3MediaStorage
from backend.infra.transactions.sqlalchemy import SQLAlchemyTransactionManager
from backend.presentation.messages.ws.connection_manager import MessageConnectionManager
from backend.presentation.messages.ws.ports import MessageConnectionManagerPort


class InfrastructureProvider(Provider):
    @provide(scope=Scope.APP)
    def app_settings(self) -> Settings:
        return settings

    @provide(scope=Scope.APP)
    def database_config(self, app_settings: Settings) -> DatabaseConfig:
        return app_settings.db

    @provide(scope=Scope.APP)
    def redis_config(self, app_settings: Settings) -> RedisConfig:
        return app_settings.redis

    @provide(scope=Scope.APP)
    def jwt_config(self, app_settings: Settings) -> JwtConfig:
        return app_settings.jwt

    @provide(scope=Scope.APP)
    def file_storage_config(self, app_settings: Settings) -> FileStorageConfig:
        return app_settings.file_storage

    @provide(scope=Scope.APP)
    def frontend_config(self, app_settings: Settings) -> FrontendConfig:
        return app_settings.frontend

    @provide(scope=Scope.APP)
    def clickhouse_config(self, app_settings: Settings) -> ClickHouseConfig:
        return app_settings.clickhouse

    @provide(scope=Scope.APP)
    async def engine(
        self,
        database_config: DatabaseConfig,
    ) -> AsyncIterator[AsyncEngine]:
        engine = create_async_engine(
            url=str(database_config.url),
            echo=database_config.echo,
            echo_pool=database_config.echo_pool,
            pool_size=database_config.pool_size,
            max_overflow=database_config.max_overflow,
            pool_pre_ping=True,
        )
        yield engine
        await engine.dispose()

    @provide(scope=Scope.APP)
    def session_factory(
        self,
        engine: AsyncEngine,
    ) -> async_sessionmaker[AsyncSession]:
        return async_sessionmaker(
            bind=engine,
            autoflush=False,
            autocommit=False,
            expire_on_commit=False,
        )

    @provide(scope=Scope.REQUEST)
    async def session(
        self,
        session_factory: async_sessionmaker[AsyncSession],
    ) -> AsyncIterator[AsyncSession]:
        async with session_factory() as session:
            yield session

    @provide(scope=Scope.SESSION)
    async def websocket_session(
        self,
        session_factory: async_sessionmaker[AsyncSession],
    ) -> AsyncIterator[AsyncSession]:
        """Provide one database session for the lifetime of a WebSocket."""
        async with session_factory() as session:
            yield session

    @provide(scope=Scope.REQUEST, provides=TransactionManager)
    def transaction_manager(self, session: AsyncSession) -> SQLAlchemyTransactionManager:
        return SQLAlchemyTransactionManager(session=session)

    @provide(scope=Scope.SESSION, provides=TransactionManager)
    def websocket_transaction_manager(self, session: AsyncSession) -> SQLAlchemyTransactionManager:
        return SQLAlchemyTransactionManager(session=session)

    @provide(scope=Scope.REQUEST, provides=UserRepositoryPort)
    def user_repository_port(self, session: AsyncSession) -> InfraUserRepository:
        return InfraUserRepository(session=session)

    @provide(scope=Scope.SESSION, provides=UserRepositoryPort)
    def websocket_user_repository(self, session: AsyncSession) -> InfraUserRepository:
        return InfraUserRepository(session=session)

    @provide(scope=Scope.REQUEST, provides=PostRepositoryPort)
    def post_repository_port(self, session: AsyncSession) -> InfraPostRepository:
        return InfraPostRepository(session=session)

    @provide(scope=Scope.REQUEST, provides=ProfileRepository)
    def profile_repository(self, session: AsyncSession) -> InfraProfileRepository:
        return InfraProfileRepository(session=session)

    @provide(scope=Scope.SESSION, provides=ProfileRepository)
    def websocket_profile_repository(self, session: AsyncSession) -> InfraProfileRepository:
        return InfraProfileRepository(session=session)

    @provide(scope=Scope.REQUEST, provides=BlockRepository)
    def block_repository(self, session: AsyncSession) -> InfraBlockRepository:
        return InfraBlockRepository(session=session)

    @provide(scope=Scope.SESSION, provides=BlockRepository)
    def websocket_block_repository(self, session: AsyncSession) -> InfraBlockRepository:
        return InfraBlockRepository(session=session)

    @provide(scope=Scope.REQUEST, provides=AuthorizationService)
    def authorization_service(self, session: AsyncSession) -> InfraAuthorizationService:
        return InfraAuthorizationService(session=session)

    @provide(scope=Scope.REQUEST, provides=AdminRepository)
    def admin_repository(self, session: AsyncSession) -> InfraAdminRepository:
        return InfraAdminRepository(session=session)

    @provide(scope=Scope.REQUEST, provides=AuditRepository)
    def audit_repository(self, session: AsyncSession) -> InfraAuditRepository:
        return InfraAuditRepository(session=session)

    @provide(scope=Scope.REQUEST, provides=CommentRepository)
    def comment_repository(self, session: AsyncSession) -> InfraCommentRepository:
        return InfraCommentRepository(session=session)

    @provide(scope=Scope.REQUEST, provides=LikeRepository)
    def like_repository(self, session: AsyncSession) -> InfraLikeRepository:
        return InfraLikeRepository(session=session)

    @provide(scope=Scope.REQUEST, provides=MessageRepository)
    def message_repository(self, session: AsyncSession) -> InfraMessageRepository:
        return InfraMessageRepository(session=session)

    @provide(scope=Scope.SESSION, provides=MessageRepository)
    def websocket_message_repository(self, session: AsyncSession) -> InfraMessageRepository:
        return InfraMessageRepository(session=session)

    @provide(scope=Scope.REQUEST, provides=FriendRepository)
    def friend_repository(self, session: AsyncSession) -> InfraFriendRepository:
        return InfraFriendRepository(session=session)

    @provide(scope=Scope.SESSION, provides=FriendRepository)
    def websocket_friend_repository(self, session: AsyncSession) -> InfraFriendRepository:
        return InfraFriendRepository(session=session)

    @provide(scope=Scope.REQUEST, provides=OutboxRepository)
    def outbox_repository(self, session: AsyncSession) -> InfraOutboxRepository:
        return InfraOutboxRepository(session=session)

    @provide(scope=Scope.SESSION, provides=OutboxRepository)
    def websocket_outbox_repository(self, session: AsyncSession) -> InfraOutboxRepository:
        return InfraOutboxRepository(session=session)

    @provide(scope=Scope.REQUEST)
    async def clickhouse_client(
        self,
        clickhouse_config: ClickHouseConfig,
    ) -> AsyncIterator[ClickHouseAnalyticsClient]:
        client = await ClickHouseAnalyticsClient.create(clickhouse_config)
        await client.init_schema()
        yield client
        await client.close()

    @provide(scope=Scope.REQUEST, provides=AnalyticsRepository)
    def analytics_repository(
        self,
        clickhouse_client: ClickHouseAnalyticsClient,
    ) -> ClickHouseAnalyticsRepository:
        return ClickHouseAnalyticsRepository(clickhouse_client)

    @provide(scope=Scope.APP, provides=PendingTokenStore)
    def pending_token_store(self, redis_config: RedisConfig) -> RedisPendingTokenStore:
        return RedisPendingTokenStore(redis_config=redis_config)

    @provide(scope=Scope.APP, provides=PasswordHasher)
    def password_hasher(self) -> BcryptPasswordHasher:
        return BcryptPasswordHasher()

    @provide(scope=Scope.APP, provides=AuthTokenService)
    def auth_token_service(self, jwt_config: JwtConfig) -> JwtAuthTokenService:
        return JwtAuthTokenService(jwt_config=jwt_config)

    @provide(scope=Scope.APP)
    def message_event_broker(
        self,
        redis_config: RedisConfig,
    ) -> MessageEventBroker:
        return RedisMessageEventBroker(redis_config)

    @provide(scope=Scope.APP, provides=MessageConnectionManagerPort)
    async def message_connection_manager(
        self,
        event_broker: MessageEventBroker,
    ) -> AsyncIterator[MessageConnectionManagerPort]:
        manager = MessageConnectionManager(event_broker)
        yield manager
        await manager.close()

    @provide(scope=Scope.REQUEST, provides=NotificationSender)
    def notification_sender(
        self,
        outbox_repository: OutboxRepository,
        transaction_manager: TransactionManager,
    ) -> OutboxNotificationSender:
        return OutboxNotificationSender(
            outbox_repository=outbox_repository,
            transaction_manager=transaction_manager,
        )

    @provide(scope=Scope.REQUEST, provides=FileUploadService)
    def file_upload_service(
        self,
        media_storage: MediaStorage,
    ) -> FileUploadService:
        return create_file_upload_service(media_storage)

    @provide(scope=Scope.REQUEST, provides=MediaStorage)
    def media_storage(
        self,
        session: AsyncSession,
        file_storage_config: FileStorageConfig,
    ) -> MediaStorage:
        return S3MediaStorage(
            session,
            bucket_name=file_storage_config.s3_bucket_name,
            region_name=file_storage_config.s3_region_name,
            endpoint_url=file_storage_config.s3_endpoint_url,
            access_key_id=file_storage_config.s3_access_key_id,
            secret_access_key=file_storage_config.s3_secret_access_key,
            key_prefix=file_storage_config.s3_key_prefix,
            max_size_bytes=file_storage_config.max_file_size_bytes,
        )


class ApplicationProvider(Provider):
    @provide(scope=Scope.REQUEST)
    def messaging_use_case(
        self,
        message_repository: MessageRepository,
        outbox_repository: OutboxRepository,
        transaction_manager: TransactionManager,
        media_storage: MediaStorage,
        profile_repository: ProfileRepository,
        friend_repository: FriendRepository,
        block_repository: BlockRepository,
    ) -> MessagingUseCase:
        return MessagingUseCase(
            message_repository,
            outbox_repository,
            transaction_manager,
            media_storage,
            profile_repository,
            friend_repository,
            block_repository,
        )

    @provide(scope=Scope.SESSION)
    def websocket_messaging_use_case(
        self,
        message_repository: MessageRepository,
        outbox_repository: OutboxRepository,
        transaction_manager: TransactionManager,
        profile_repository: ProfileRepository,
        friend_repository: FriendRepository,
        block_repository: BlockRepository,
    ) -> MessagingUseCase:
        return MessagingUseCase(
            message_repository,
            outbox_repository,
            transaction_manager,
            profile_repository=profile_repository,
            friend_repository=friend_repository,
            block_repository=block_repository,
        )
    @provide(scope=Scope.REQUEST)
    def login_use_case(
        self,
        user_repository: UserRepositoryPort,
        password_hasher: PasswordHasher,
        token_service: AuthTokenService,
    ) -> LoginUseCase:
        return LoginUseCase(user_repository, password_hasher, token_service)

    @provide(scope=Scope.REQUEST)
    def current_user_use_case(
        self,
        user_repository: UserRepositoryPort,
        token_service: AuthTokenService,
    ) -> GetCurrentUserUseCase:
        return GetCurrentUserUseCase(user_repository, token_service)

    @provide(scope=Scope.REQUEST)
    def touch_user_activity_use_case(
        self,
        user_repository: UserRepositoryPort,
    ) -> TouchUserActivityUseCase:
        return TouchUserActivityUseCase(user_repository)

    @provide(scope=Scope.REQUEST)
    def refresh_token_use_case(
        self,
        token_service: AuthTokenService,
    ) -> RefreshTokenUseCase:
        return RefreshTokenUseCase(token_service)

    @provide(scope=Scope.REQUEST)
    def registration_confirmation_use_case(
        self,
        user_repository: UserRepositoryPort,
        token_store: PendingTokenStore,
        notification_sender: NotificationSender,
    ) -> RequestRegistrationConfirmationUseCase:
        return RequestRegistrationConfirmationUseCase(
            user_repository,
            token_store,
            notification_sender,
        )

    @provide(scope=Scope.REQUEST)
    def confirm_registration_use_case(
        self,
        user_repository: UserRepositoryPort,
        token_store: PendingTokenStore,
        token_service: AuthTokenService,
        transaction_manager: TransactionManager,
    ) -> ConfirmRegistrationUseCase:
        return ConfirmRegistrationUseCase(
            user_repository,
            token_store,
            token_service,
            transaction_manager,
        )

    @provide(scope=Scope.REQUEST)
    def register_user_use_case(
        self,
        user_repository: UserRepositoryPort,
        password_hasher: PasswordHasher,
        token_store: PendingTokenStore,
        notification_sender: NotificationSender,
        outbox_repository: OutboxRepository,
        transaction_manager: TransactionManager,
    ) -> RegisterUserUseCase:
        return RegisterUserUseCase(
            user_repository,
            password_hasher,
            token_store,
            notification_sender,
            outbox_repository,
            transaction_manager,
        )

    @provide(scope=Scope.REQUEST)
    def password_reset_request_use_case(
        self,
        user_repository: UserRepositoryPort,
        token_store: PendingTokenStore,
        notification_sender: NotificationSender,
    ) -> RequestPasswordResetUseCase:
        return RequestPasswordResetUseCase(
            user_repository,
            token_store,
            notification_sender,
        )

    @provide(scope=Scope.REQUEST)
    def reset_password_use_case(
        self,
        user_repository: UserRepositoryPort,
        token_store: PendingTokenStore,
        password_hasher: PasswordHasher,
        token_service: AuthTokenService,
        transaction_manager: TransactionManager,
    ) -> ResetPasswordUseCase:
        return ResetPasswordUseCase(
            user_repository,
            token_store,
            password_hasher,
            token_service,
            transaction_manager,
        )

    @provide(scope=Scope.REQUEST)
    def feed_use_case(self, post_repository: PostRepositoryPort) -> GetFeedUseCase:
        return GetFeedUseCase(post_repository)

    @provide(scope=Scope.REQUEST)
    def create_post_use_case(
        self,
        post_repository: PostRepositoryPort,
        outbox_repository: OutboxRepository,
        transaction_manager: TransactionManager,
        file_upload_service: FileUploadService,
    ) -> CreatePostUseCase:
        return CreatePostUseCase(
            post_repository,
            outbox_repository,
            transaction_manager,
            file_upload_service,
        )

    @provide(scope=Scope.REQUEST)
    def update_post_use_case(
        self,
        post_repository: PostRepositoryPort,
        transaction_manager: TransactionManager,
        file_upload_service: FileUploadService,
    ) -> UpdatePostUseCase:
        return UpdatePostUseCase(post_repository, transaction_manager, file_upload_service)

    @provide(scope=Scope.REQUEST)
    def delete_post_use_case(
        self,
        post_repository: PostRepositoryPort,
        transaction_manager: TransactionManager,
    ) -> DeletePostUseCase:
        return DeletePostUseCase(post_repository, transaction_manager)

    @provide(scope=Scope.REQUEST)
    def remove_post_image_use_case(
        self,
        post_repository: PostRepositoryPort,
        transaction_manager: TransactionManager,
    ) -> RemovePostImageUseCase:
        return RemovePostImageUseCase(post_repository, transaction_manager)

    @provide(scope=Scope.REQUEST)
    def profile_use_case(
        self,
        profile_repository: ProfileRepository,
        post_repository: PostRepositoryPort,
        friend_repository: FriendRepository,
        block_repository: BlockRepository,
    ) -> GetProfileUseCase:
        return GetProfileUseCase(profile_repository, post_repository, friend_repository, block_repository)

    @provide(scope=Scope.REQUEST)
    def update_profile_use_case(
        self,
        profile_repository: ProfileRepository,
        transaction_manager: TransactionManager,
    ) -> UpdateProfileUseCase:
        return UpdateProfileUseCase(profile_repository, transaction_manager)

    @provide(scope=Scope.REQUEST)
    def upload_avatar_use_case(
        self,
        profile_repository: ProfileRepository,
        transaction_manager: TransactionManager,
        media_storage: MediaStorage,
    ) -> UploadAvatarUseCase:
        return UploadAvatarUseCase(
            profile_repository,
            transaction_manager,
            media_storage,
        )

    @provide(scope=Scope.REQUEST)
    def remove_avatar_use_case(
        self,
        profile_repository: ProfileRepository,
        transaction_manager: TransactionManager,
    ) -> RemoveAvatarUseCase:
        return RemoveAvatarUseCase(profile_repository, transaction_manager)

    @provide(scope=Scope.REQUEST)
    def avatar_history_use_case(
        self,
        profile_repository: ProfileRepository,
    ) -> GetAvatarHistoryUseCase:
        return GetAvatarHistoryUseCase(profile_repository)

    @provide(scope=Scope.REQUEST)
    def select_avatar_use_case(
        self,
        profile_repository: ProfileRepository,
        transaction_manager: TransactionManager,
    ) -> SelectAvatarUseCase:
        return SelectAvatarUseCase(profile_repository, transaction_manager)

    @provide(scope=Scope.REQUEST)
    def profile_photos_use_case(
        self,
        profile_repository: ProfileRepository,
    ) -> GetProfilePhotosUseCase:
        return GetProfilePhotosUseCase(profile_repository)

    @provide(scope=Scope.REQUEST)
    def create_photo_album_use_case(
        self,
        profile_repository: ProfileRepository,
        transaction_manager: TransactionManager,
    ) -> CreatePhotoAlbumUseCase:
        return CreatePhotoAlbumUseCase(profile_repository, transaction_manager)

    @provide(scope=Scope.REQUEST)
    def upload_profile_photo_use_case(
        self,
        profile_repository: ProfileRepository,
        transaction_manager: TransactionManager,
        file_upload_service: FileUploadService,
    ) -> UploadProfilePhotoUseCase:
        return UploadProfilePhotoUseCase(
            profile_repository,
            transaction_manager,
            file_upload_service,
        )

    @provide(scope=Scope.REQUEST)
    def delete_profile_photo_use_case(
        self,
        profile_repository: ProfileRepository,
        outbox_repository: OutboxRepository,
        transaction_manager: TransactionManager,
    ) -> DeleteProfilePhotoUseCase:
        return DeleteProfilePhotoUseCase(
            profile_repository,
            outbox_repository,
            transaction_manager,
        )

    @provide(scope=Scope.REQUEST)
    def create_comment_use_case(
        self,
        comment_repository: CommentRepository,
        outbox_repository: OutboxRepository,
        transaction_manager: TransactionManager,
        block_repository: BlockRepository,
    ) -> CreateCommentUseCase:
        return CreateCommentUseCase(comment_repository, outbox_repository, transaction_manager, block_repository)

    @provide(scope=Scope.REQUEST)
    def comments_use_case(self, comment_repository: CommentRepository) -> GetCommentsUseCase:
        return GetCommentsUseCase(comment_repository)

    @provide(scope=Scope.REQUEST)
    def update_comment_use_case(
        self,
        comment_repository: CommentRepository,
        transaction_manager: TransactionManager,
    ) -> UpdateCommentUseCase:
        return UpdateCommentUseCase(comment_repository, transaction_manager)

    @provide(scope=Scope.REQUEST)
    def delete_comment_use_case(
        self,
        comment_repository: CommentRepository,
        transaction_manager: TransactionManager,
        authorization: AuthorizationService,
        audit_repository: AuditRepository,
    ) -> DeleteCommentUseCase:
        return DeleteCommentUseCase(comment_repository, transaction_manager, authorization, audit_repository)

    @provide(scope=Scope.REQUEST)
    def admin_rbac_use_case(
        self,
        admin_repository: AdminRepository,
        authorization: AuthorizationService,
        transaction_manager: TransactionManager,
    ) -> AdminRbacUseCase:
        return AdminRbacUseCase(admin_repository, authorization, transaction_manager)

    @provide(scope=Scope.REQUEST)
    def toggle_post_like_use_case(
        self,
        like_repository: LikeRepository,
        outbox_repository: OutboxRepository,
        transaction_manager: TransactionManager,
    ) -> TogglePostLikeUseCase:
        return TogglePostLikeUseCase(like_repository, outbox_repository, transaction_manager)

    @provide(scope=Scope.REQUEST)
    def toggle_comment_like_use_case(
        self,
        like_repository: LikeRepository,
        outbox_repository: OutboxRepository,
        transaction_manager: TransactionManager,
    ) -> ToggleCommentLikeUseCase:
        return ToggleCommentLikeUseCase(like_repository, outbox_repository, transaction_manager)

    @provide(scope=Scope.REQUEST)
    def analytics_summary_use_case(
        self,
        analytics_repository: AnalyticsRepository,
    ) -> GetAnalyticsSummaryUseCase:
        return GetAnalyticsSummaryUseCase(analytics_repository)

    @provide(scope=Scope.REQUEST)
    def get_friends_use_case(self, friend_repository: FriendRepository) -> GetFriendsUseCase:
        return GetFriendsUseCase(friend_repository)

    @provide(scope=Scope.REQUEST)
    def friend_recommendations_use_case(
        self, friend_repository: FriendRepository, block_repository: BlockRepository
    ) -> GetFriendRecommendationsUseCase:
        return GetFriendRecommendationsUseCase(friend_repository, block_repository)

    @provide(scope=Scope.REQUEST)
    def add_friend_use_case(
        self,
        friend_repository: FriendRepository,
        outbox_repository: OutboxRepository,
        transaction_manager: TransactionManager,
        profile_repository: ProfileRepository,
        block_repository: BlockRepository,
    ) -> AddFriendUseCase:
        return AddFriendUseCase(
            friend_repository,
            outbox_repository,
            transaction_manager,
            block_repository,
            profile_repository,
        )

    @provide(scope=Scope.REQUEST)
    def block_user_use_case(
        self,
        block_repository: BlockRepository,
        friend_repository: FriendRepository,
        transaction_manager: TransactionManager,
        audit_repository: AuditRepository,
    ) -> BlockUserUseCase:
        return BlockUserUseCase(block_repository, friend_repository, transaction_manager, audit_repository)

    @provide(scope=Scope.REQUEST)
    def unblock_user_use_case(
        self, block_repository: BlockRepository, transaction_manager: TransactionManager
    ) -> UnblockUserUseCase:
        return UnblockUserUseCase(block_repository, transaction_manager)

    @provide(scope=Scope.REQUEST)
    def remove_friend_use_case(
        self,
        friend_repository: FriendRepository,
        outbox_repository: OutboxRepository,
        transaction_manager: TransactionManager,
    ) -> RemoveFriendUseCase:
        return RemoveFriendUseCase(friend_repository, outbox_repository, transaction_manager)

    @provide(scope=Scope.REQUEST)
    def cancel_subscription_use_case(
        self,
        friend_repository: FriendRepository,
        outbox_repository: OutboxRepository,
        transaction_manager: TransactionManager,
    ) -> CancelSubscriptionUseCase:
        return CancelSubscriptionUseCase(friend_repository, outbox_repository, transaction_manager)
