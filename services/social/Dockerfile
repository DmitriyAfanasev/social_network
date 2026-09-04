FROM golang:1.27-alpine AS build

WORKDIR /src

COPY libs/platform/go.mod libs/platform/go.sum ./libs/platform/
COPY services/social/go.mod ./services/social/
COPY libs/platform ./libs/platform
COPY services/social ./services/social

WORKDIR /src/services/social
RUN GOWORK=off go mod download
RUN CGO_ENABLED=0 GOWORK=off go build -o /out/social ./cmd/social

FROM alpine:3.22

RUN addgroup -S app && adduser -S -G app app
COPY --from=build /out/social /usr/local/bin/social

USER app
EXPOSE 8103
ENTRYPOINT ["/usr/local/bin/social"]
