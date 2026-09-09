FROM golang:1.27-alpine AS build

WORKDIR /src

COPY libs/platform/go.mod libs/platform/go.sum ./libs/platform/
COPY services/media/go.mod ./services/media/
COPY libs/platform ./libs/platform
COPY services/media ./services/media

WORKDIR /src/services/media
RUN GOWORK=off go mod download
RUN CGO_ENABLED=0 GOWORK=off go build -o /out/media ./cmd/media
RUN CGO_ENABLED=0 GOWORK=off go build -o /out/media-image-worker ./cmd/media-image-worker

FROM alpine:3.22

RUN addgroup -S app && adduser -S -G app app
COPY --from=build /out/media /usr/local/bin/media
COPY --from=build /out/media-image-worker /usr/local/bin/media-image-worker

USER app
EXPOSE 8105
ENTRYPOINT ["/usr/local/bin/media"]
