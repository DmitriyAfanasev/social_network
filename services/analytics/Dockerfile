FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.work go.work.sum ./
COPY libs/platform/go.mod libs/platform/go.sum libs/platform/
COPY services/analytics/go.mod services/analytics/
RUN mkdir -p services/analytics/internal services/analytics/cmd
COPY services/analytics services/analytics
COPY libs/platform libs/platform

WORKDIR /src/services/analytics
RUN GOWORK=off go mod download
RUN CGO_ENABLED=0 GOWORK=off go build -o /out/analytics ./cmd/analytics

FROM alpine:3.22
RUN adduser -D -H app
COPY --from=build /out/analytics /analytics
USER app
ENTRYPOINT ["/analytics"]
