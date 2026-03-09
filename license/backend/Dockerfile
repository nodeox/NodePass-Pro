# syntax=docker/dockerfile:1.7

FROM golang:1.23-alpine AS builder
WORKDIR /src

RUN apk add --no-cache git build-base

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux \
  go build -trimpath -ldflags="-s -w" -o /out/license-server ./cmd/server

FROM alpine:3.20
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata wget \
  && mkdir -p /app/data /app/uploads/releases \
  && chmod -R 755 /app

COPY --from=builder /out/license-server /app/license-server

ENV GIN_MODE=release \
  SERVER_PORT=8091 \
  DB_DRIVER=sqlite \
  DB_DSN=/app/data/license-unified.db \
  RELEASE_UPLOAD_DIR=/app/uploads/releases

EXPOSE 8091

HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8091/health || exit 1

ENTRYPOINT ["/app/license-server"]
