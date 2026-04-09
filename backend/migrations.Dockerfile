# Build stage (optional, if we needed to bundle a specific version of migrate)
# But we can use the community image directly or build from scratch
FROM ghcr.io/opennsw/migrate:latest AS binary

# Migration image
FROM alpine:3.18

# Install postgres-client and CA certs
RUN apk add --no-cache ca-certificates postgresql-client

# OPENSHIFT COMPLIANCE: Use UID 1001 and Group 0
# OpenShift will override the UID but keeps GID 0.
RUN adduser -D -u 1001 -G root -s /bin/false migrate

WORKDIR /migrations

# Ensure GID 0 has access to the app directory
RUN chgrp -R 0 /migrations && \
    chmod -R g+rwX /migrations

USER 1001


# Copy migration binary from official image
COPY --from=migrate/migrate /usr/bin/migrate /usr/local/bin/migrate

# Copy SQL scripts
COPY backend/internal/database/migrations/*.sql ./

# Default entrypoint
ENTRYPOINT ["migrate"]
CMD ["--help"]
