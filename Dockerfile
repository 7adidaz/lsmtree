# Start from the official Golang image for building
FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o lsm-tree main.go

FROM debian:bookworm-slim

WORKDIR /app

COPY --from=builder /app/lsm-tree .

EXPOSE 8080

RUN mkdir -p /app/data

CMD ["./lsm-tree"]
