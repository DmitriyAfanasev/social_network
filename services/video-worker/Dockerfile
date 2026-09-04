FROM golang:1.27-alpine AS build
WORKDIR /src
COPY libs/platform/go.mod libs/platform/go.sum libs/platform/
COPY services/video-worker/go.mod services/video-worker/go.sum services/video-worker/
COPY libs/platform libs/platform
COPY services/video-worker services/video-worker
WORKDIR /src/services/video-worker
RUN GOWORK=off go mod download
RUN CGO_ENABLED=0 GOWORK=off go build -trimpath -ldflags="-s -w" -o /video-worker ./cmd/video-worker

FROM alpine:3.22

RUN apk add --no-cache ffmpeg

COPY --from=build /video-worker /usr/local/bin/video-worker
ENTRYPOINT ["/usr/local/bin/video-worker"]
