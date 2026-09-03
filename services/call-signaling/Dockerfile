FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /call-signaling ./cmd/call-signaling

FROM alpine:3.22
RUN adduser -D -H appuser
COPY --from=build /call-signaling /usr/local/bin/call-signaling
USER appuser
EXPOSE 8001
ENTRYPOINT ["/usr/local/bin/call-signaling"]
