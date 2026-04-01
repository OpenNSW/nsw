# Use the official Golang image as a builder
FROM golang:1.25-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Ensure the bucket exists and we build for the correct cluster architecture
RUN mkdir -p /app/bucket && \
    CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

FROM debian:bookworm-slim

LABEL org.opencontainers.image.source="https://github.com/OpenNSW/nsw"
LABEL org.opencontainers.image.description="NSW Backend Service"

# Install certs and create a non-root user
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -r -u 1001 -g 0 -s /bin/false -d /app appuser

WORKDIR /app

# Copy the binary and assets
COPY --from=builder /out/server /app/server
COPY --from=builder /app/configs /app/configs
COPY --from=builder /app/bucket /app/bucket
COPY --from=builder /app/internal/database/migrations /app/migrations

# OPENSHIFT COMPLIANCE:
# OpenShift runs as a random UID in Group 0. We must ensure Group 0 
# has write access to /app and the bucket for the app to function.
RUN chgrp -R 0 /app && \
    chmod -R g+rwX /app && \
    chmod +x /app/server

# Switch to the non-root user (OpenShift will override this UID, but will keep GID 0)
USER 1001

EXPOSE 8080


CMD ["/app/server"]