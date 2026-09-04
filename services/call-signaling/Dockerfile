FROM golang:1.27-alpine AS build
WORKDIR /src
COPY libs/platform/go.mod libs/platform/go.sum libs/platform/
COPY services/call-signaling/go.mod services/call-signaling/go.sum services/call-signaling/
COPY libs/platform libs/platform
COPY services/call-signaling services/call-signaling
WORKDIR /src/services/call-signaling
RUN GOWORK=off go mod download
RUN CGO_ENABLED=0 GOWORK=off go build -trimpath -ldflags="-s -w" -o /call-signaling ./cmd/call-signaling

FROM alpine:3.22
RUN adduser -D -H appuser
COPY --from=build /call-signaling /usr/local/bin/call-signaling
USER appuser
EXPOSE 8001
ENTRYPOINT ["/usr/local/bin/call-signaling"]
