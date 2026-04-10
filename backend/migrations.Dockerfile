# Use the official migrate image to extract the binary
# Always pin a specific version for production immutability
FROM migrate/migrate:v4.17.0 AS migrate-bin

# Build the execution image
FROM alpine:3.18

# Install postgres-client (for pg_isready checks) and CA certs
RUN apk add --no-cache ca-certificates postgresql-client

# OPENSHIFT COMPLIANCE: Use UID 1001 and Group 0
# OpenShift will override the UID dynamically, but it relies on GID 0 permissions
RUN adduser -D -u 1001 -G root -s /bin/false migrate

WORKDIR /migrations

# Ensure the root group (GID 0) has access to the app directory
RUN chgrp -R 0 /migrations && \
    chmod -R g+rwX /migrations

# Copy migration binary from the builder stage
# (Note: The official image places it at /migrate, not /usr/bin/migrate)
COPY --from=migrate-bin /migrate /usr/local/bin/migrate

# Copy SQL scripts (Assuming Docker build context is the repository root)
COPY backend/internal/database/migrations/*.sql ./

# Set to the non-root user
USER 1001