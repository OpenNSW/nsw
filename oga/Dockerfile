# Build stage: OGA service (Go + CGO for SQLite)
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/oga-server ./cmd/server

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates

WORKDIR /app

# SQLite driver needs libc and sqlite; use same libs as build or install sqlite
RUN apk add --no-cache sqlite-libs

COPY --from=builder /app/oga-server /app/oga-server
COPY data/forms /app/forms

EXPOSE 8081

ENTRYPOINT ["/app/oga-server"]
