#!/usr/bin/env bash
set -euo pipefail

topics=(
  "user.registered"
  "post.created"
  "post.like_toggled"
  "comment.created"
  "profile_photo.deleted"
  "friend.requested"
  "friend.accepted"
  "friend.removed"
  "friend.request_cancelled"
  "comment.like_toggled"
  "message.sent"
  "message.edited"
  "message.deleted"
  "message.media_removed"
  "message.read"
  "analytics.dlq"
  "email.registration_confirmation_requested"
  "email.password_reset_requested"
  "video.transcode.requested"
  "video.transcode.completed"
)

echo "Waiting for Kafka..."
until docker compose exec -T kafka /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 \
  --list >/dev/null 2>&1; do
  sleep 1
done

for topic in "${topics[@]}"; do
  docker compose exec -T kafka /opt/kafka/bin/kafka-topics.sh \
    --bootstrap-server localhost:9092 \
    --create \
    --if-not-exists \
    --topic "$topic" \
    --partitions 1 \
    --replication-factor 1
done

echo "Kafka topics are ready."
