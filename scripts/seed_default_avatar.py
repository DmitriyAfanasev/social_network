"""Загружает системный аватар в MinIO и создаёт его запись в таблице media."""

import asyncio
import hashlib
from pathlib import Path

import boto3
from sqlalchemy import select
from sqlalchemy.ext.asyncio import async_sessionmaker, create_async_engine

from backend.infra.config import settings
from backend.infra.models.sqlalchemy import Media


OBJECT_KEY = "system/default-avatar.svg"
ASSET_PATH = Path(__file__).resolve().parent.parent / "backend" / "assets" / "default_avatar.svg"


async def seed_default_avatar() -> None:
    """Загружает дефолтный аватар идемпотентно."""
    content = ASSET_PATH.read_bytes()
    client = boto3.client(
        "s3",
        endpoint_url=settings.file_storage.s3_endpoint_url,
        region_name=settings.file_storage.s3_region_name,
        aws_access_key_id=settings.file_storage.s3_access_key_id.get_secret_value(),
        aws_secret_access_key=settings.file_storage.s3_secret_access_key.get_secret_value(),
    )
    client.put_object(
        Bucket=settings.file_storage.s3_bucket_name,
        Key=OBJECT_KEY,
        Body=content,
        ContentType="image/svg+xml",
    )

    engine = create_async_engine(str(settings.db.url))
    session_factory = async_sessionmaker(engine, expire_on_commit=False)
    async with session_factory() as session:
        media = await session.scalar(select(Media).where(Media.object_key == OBJECT_KEY))
        if media is None:
            session.add(
                Media(
                    object_key=OBJECT_KEY,
                    bucket=settings.file_storage.s3_bucket_name,
                    original_filename="default_avatar.svg",
                    content_type="image/svg+xml",
                    media_type="image",
                    size=len(content),
                    checksum=hashlib.sha256(content).hexdigest(),
                )
            )
        await session.commit()
    await engine.dispose()


if __name__ == "__main__":
    asyncio.run(seed_default_avatar())
