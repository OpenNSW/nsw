FROM golang:1.25-alpine AS builder
RUN apk add --no-cache gcc musl-dev sqlite-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o bin/oga ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache sqlite-libs ca-certificates
WORKDIR /app
COPY --from=builder /app/bin/oga /app/oga
COPY --from=builder /app/data /app/data
EXPOSE 8081
CMD ["/app/oga"]
