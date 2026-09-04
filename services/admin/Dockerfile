FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.work go.work.sum ./
COPY libs/platform/go.mod libs/platform/go.sum libs/platform/
COPY services/admin/go.mod services/admin/go.sum services/admin/
RUN mkdir -p services/admin/internal services/admin/cmd
COPY services/admin services/admin
COPY libs/platform libs/platform

WORKDIR /src/services/admin
RUN GOWORK=off go mod download
RUN CGO_ENABLED=0 GOWORK=off go build -o /out/admin ./cmd/admin

FROM alpine:3.22
RUN adduser -D -H app
COPY --from=build /out/admin /admin
USER app
ENTRYPOINT ["/admin"]
