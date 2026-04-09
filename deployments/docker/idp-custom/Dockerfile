FROM ghcr.io/asgardeo/thunder:0.29.0

# Switch to root temporarily to modify permissions
USER root

# Replace the broken binary
COPY ./fixed-binary /opt/thunder/bin/binary

# Grant Group 0 ownership and arbitrary UID compliance across /opt/thunder
RUN chgrp -R 0 /opt/thunder && \
    chmod -R g+rwx /opt/thunder

# Switch back to the default non-root user
USER 10001