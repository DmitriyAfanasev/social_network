FROM golang:1.27-alpine AS build

WORKDIR /src

COPY libs/platform/go.mod libs/platform/go.sum ./libs/platform/
COPY services/identity/go.mod services/identity/go.sum ./services/identity/
COPY libs/platform ./libs/platform
COPY services/identity ./services/identity

WORKDIR /src/services/identity
RUN GOWORK=off go mod download
RUN CGO_ENABLED=0 GOWORK=off go build -o /out/identity ./cmd/identity

FROM alpine:3.22

RUN addgroup -S app && adduser -S -G app app
COPY --from=build /out/identity /usr/local/bin/identity

USER app
EXPOSE 8101
ENTRYPOINT ["/usr/local/bin/identity"]
