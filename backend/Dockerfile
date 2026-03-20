FROM golang:1.25-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN mkdir -p /app/bucket
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

FROM debian:bookworm-slim

LABEL org.opencontainers.image.source="https://github.com/OpenNSW/nsw"
LABEL org.opencontainers.image.description="NSW Backend API Service"

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates wget \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -r -s /bin/false -d /app appuser

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /out/server /app/server
COPY --from=builder --chown=appuser:appuser /app/bucket /app/bucket

USER appuser:appuser

# Expose the application port (configurable via SERVER_PORT env var)
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
	CMD wget -qO- http://localhost:${SERVER_PORT:-8080}/health || exit 1

# Run the binary
CMD ["/app/server"]
