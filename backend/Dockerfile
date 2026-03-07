FROM golang:1.21-bookworm

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /usr/local/bin/nodepass-backend ./cmd/server

EXPOSE 8080

CMD ["/usr/local/bin/nodepass-backend"]
