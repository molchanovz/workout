FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY .. .

ENV CGO_ENABLED=0

RUN go build ./cmd/workout/main.go

FROM alpine:3.21

WORKDIR /app


COPY --from=builder /app/main .
COPY cfg/local.toml cfg/local.toml

CMD ["./main"]
