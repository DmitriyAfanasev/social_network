FROM golang:1.27-alpine AS build

WORKDIR /src

COPY libs/platform/go.mod libs/platform/go.sum ./libs/platform/
COPY services/profiles/go.mod ./services/profiles/
COPY libs/platform ./libs/platform
COPY services/profiles ./services/profiles

WORKDIR /src/services/profiles
RUN GOWORK=off go mod download
RUN CGO_ENABLED=0 GOWORK=off go build -o /out/profiles ./cmd/profiles

FROM alpine:3.22

RUN addgroup -S app && adduser -S -G app app
COPY --from=build /out/profiles /usr/local/bin/profiles

USER app
EXPOSE 8102
ENTRYPOINT ["/usr/local/bin/profiles"]
