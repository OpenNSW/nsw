# Use the official Golang image as a builder
FROM golang:1.25-bookworm AS builder

WORKDIR /app

# Copy dependency manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary (Architecture-agnostic, handled by Docker Buildx)
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

# --- Final Runtime Stage ---
FROM debian:bookworm-slim

LABEL org.opencontainers.image.source="https://github.com/OpenNSW/nsw"
LABEL org.opencontainers.image.description="NSW Generic Backend Service"

# Install certs and create a non-root user
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -r -u 1001 -g 0 -s /bin/false -d /app appuser

WORKDIR /app

# Copy the binary and migrations (Static assets)
COPY --from=builder /out/server /app/server
COPY --from=builder /app/internal/database/migrations /app/migrations

# NOTE: /app/configs and /app/bucket are managed via Kubernetes volumes (ConfigMap/PVC)
# NOTE: Permissions (Group 0) are managed via Pod SecurityContext

USER 1001
EXPOSE 8080

CMD ["/app/server"]
