FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -o /video-worker ./cmd/video-worker

FROM jrottenberg/ffmpeg:6-alpine
COPY --from=build /video-worker /usr/local/bin/video-worker
ENTRYPOINT ["/usr/local/bin/video-worker"]
