FROM golang:1.25-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /out/oga ./cmd/server

FROM debian:bookworm-slim

LABEL org.opencontainers.image.source="https://github.com/OpenNSW/nsw"
LABEL org.opencontainers.image.description="NSW OGA Backend Service"

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates wget \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -r -s /bin/false -d /app appuser

WORKDIR /app

COPY --from=builder /out/oga /app/oga
COPY --from=builder /app/data/forms /app/data/forms

# Runtime settings
USER appuser

# Default port (override via OGA_PORT env var)
EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:${OGA_PORT:-8081}/health || exit 1

CMD ["/app/oga"]
