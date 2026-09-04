FROM golang:1.27-alpine AS build

WORKDIR /src

COPY libs/platform/go.mod libs/platform/go.sum ./libs/platform/
COPY services/messaging/go.mod ./services/messaging/
COPY libs/platform ./libs/platform
COPY services/messaging ./services/messaging

WORKDIR /src/services/messaging
RUN GOWORK=off go mod download
RUN CGO_ENABLED=0 GOWORK=off go build -o /out/messaging ./cmd/messaging

FROM alpine:3.22

RUN addgroup -S app && adduser -S -G app app
COPY --from=build /out/messaging /usr/local/bin/messaging

USER app
EXPOSE 8106
ENTRYPOINT ["/usr/local/bin/messaging"]
