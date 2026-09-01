from backend.application.ports.file_upload_service import FileUploadService
from backend.infra.config import FileStorageConfig
from backend.infra.storage.s3_file_upload_service import S3FileUploadService


def create_file_upload_service(file_storage_config: FileStorageConfig) -> FileUploadService:
    return S3FileUploadService(
        bucket_name=file_storage_config.s3_bucket_name,
        region_name=file_storage_config.s3_region_name,
        endpoint_url=file_storage_config.s3_endpoint_url,
        access_key_id=file_storage_config.s3_access_key_id,
        secret_access_key=file_storage_config.s3_secret_access_key,
        public_base_url=file_storage_config.s3_public_base_url,
        key_prefix=file_storage_config.s3_key_prefix,
        max_size_bytes=file_storage_config.max_file_size_bytes,
    )
