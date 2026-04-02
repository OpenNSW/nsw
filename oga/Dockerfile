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
  && apt-get install -y --no-install-recommends ca-certificates \
  && rm -rf /var/lib/apt/lists/* \
  && useradd -r -s /bin/false -d /app appuser \
  && mkdir -p /app /data \
  && chown -R appuser:appuser /data /app

WORKDIR /app

COPY --from=builder /out/oga /app/oga
COPY --from=builder /app/data/forms /app/data/forms

# Runtime settings
USER appuser

# Default port (override via OGA_PORT env var)
EXPOSE 8081


USER 1001

CMD ["/app/oga"]
