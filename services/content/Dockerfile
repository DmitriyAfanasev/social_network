FROM golang:1.27-alpine AS build

WORKDIR /src

COPY libs/platform/go.mod libs/platform/go.sum ./libs/platform/
COPY services/content/go.mod ./services/content/
COPY libs/platform ./libs/platform
COPY services/content ./services/content

WORKDIR /src/services/content
RUN GOWORK=off go mod download
RUN CGO_ENABLED=0 GOWORK=off go build -o /out/content ./cmd/content

FROM alpine:3.22

RUN addgroup -S app && adduser -S -G app app
COPY --from=build /out/content /usr/local/bin/content

USER app
EXPOSE 8104
ENTRYPOINT ["/usr/local/bin/content"]
