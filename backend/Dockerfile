# ==============================================================================
# Build Stage
# ==============================================================================
FROM golang:1.25-alpine AS builder

# Install build dependencies (Make and Git are required for the Makefile targets)
RUN apk add --no-cache make git

WORKDIR /app

# Copy dependency definitions first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Ensure bucket directory exists even when repository has no tracked bucket files
RUN mkdir -p /app/bucket

# Build the binary using the Makefile 'build-linux' target
# This ensures consistency with local builds
RUN make build-linux

# ==============================================================================
# Runtime Stage
# ==============================================================================
# Use distroless for a minimal, secure production image
FROM gcr.io/distroless/static-debian12

LABEL org.opencontainers.image.source="https://github.com/OpenNSW/nsw"
LABEL org.opencontainers.image.description="NSW Backend API Service"

WORKDIR /app

# Copy the binary from the builder stage
# The 'linux' target outputs to bin/server-linux
COPY --from=builder /app/bin/server-linux /app/server
COPY --from=builder --chown=nonroot:nonroot /app/bucket /app/bucket

# Run as non-root user (distroless provides uid 65534)
USER nonroot:nonroot

# Expose the application port (configurable via SERVER_PORT env var)
EXPOSE 8080

# Run the binary
CMD ["/app/server"]
