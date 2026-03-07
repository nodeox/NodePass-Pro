FROM golang:1.21-bookworm

WORKDIR /app

ARG APP_VERSION=dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -ldflags "-X nodepass-pro/backend/internal/version.Version=${APP_VERSION}" -o /usr/local/bin/nodepass-backend ./cmd/server \
    && go build -ldflags "-X nodepass-pro/backend/internal/version.Version=${APP_VERSION}" -o /usr/local/bin/nodepass-admin-bootstrap ./cmd/admin-bootstrap

LABEL org.opencontainers.image.version="${APP_VERSION}"

EXPOSE 8080

CMD ["/usr/local/bin/nodepass-backend"]
