FROM golang:1.27-alpine AS build

WORKDIR /src

COPY libs/platform/go.mod libs/platform/go.sum ./libs/platform/
COPY services/gateway/go.mod services/gateway/go.sum ./services/gateway/
COPY libs/platform ./libs/platform
COPY services/gateway ./services/gateway

WORKDIR /src/services/gateway
RUN GOWORK=off go mod download
RUN CGO_ENABLED=0 GOWORK=off go build -o /out/gateway ./cmd/gateway

FROM alpine:3.22

RUN addgroup -S app && adduser -S -G app app
COPY --from=build /out/gateway /usr/local/bin/gateway

USER app
EXPOSE 8000
ENTRYPOINT ["/usr/local/bin/gateway"]
