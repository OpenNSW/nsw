# Build the migrate binary from source using the latest secure Go 1.25 compiler
FROM golang:1.25-bookworm AS builder

# Build migrate statically with PostgreSQL support only
RUN CGO_ENABLED=0 go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1

# Build the final execution image
FROM alpine:3.21

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
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate

# Copy SQL scripts (Assuming Docker build context is the repository root)
COPY backend/internal/database/migrations/*.sql ./

# Set to the non-root user
USER 1001