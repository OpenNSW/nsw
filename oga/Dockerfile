# Use the official Golang image as a builder
FROM golang:1.25-bookworm AS builder

WORKDIR /app

# Copy go.mod/sum from the oga directory
COPY oga/go.mod oga/go.sum ./
RUN go mod download

# Copy the oga source code
COPY oga/ .

# Build for the cluster architecture (linux/amd64)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/oga ./cmd/server

FROM debian:bookworm-slim

LABEL org.opencontainers.image.source="https://github.com/OpenNSW/nsw"
LABEL org.opencontainers.image.description="NSW OGA Backend Service"

# Install certs and create a non-root user belonging to group 0
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates wget \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -r -u 1001 -g 0 -s /bin/false -d /app appuser

WORKDIR /app
# Create the structure AND ensure it's not empty
RUN mkdir -p /app/repository/resources/security /app/repository/database && \
    touch /app/repository/resources/security/.placeholder && \
    touch /app/repository/database/.placeholder

# Copy binary and assets
COPY --from=builder /out/oga /app/oga
COPY --from=builder /app/data/forms /app/data/forms

# OPENSHIFT COMPLIANCE: GID 0 Strategy
USER root
RUN chgrp -R 0 /app && \
    chmod -R g=u /app && \
    chmod -R g+rwX /app
USER 1001

CMD ["/app/oga"]