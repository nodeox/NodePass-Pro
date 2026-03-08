FROM golang:1.24-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.appVersion=0.2.0" -o /usr/local/bin/license-center ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /usr/local/bin/license-center /usr/local/bin/license-center
COPY configs/config.yaml /app/configs/config.yaml
EXPOSE 8090
CMD ["/usr/local/bin/license-center", "--config", "/app/configs/config.yaml"]
