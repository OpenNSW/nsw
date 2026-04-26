FROM ghcr.io/asgardeo/thunder:0.34.0

# Switch to root temporarily to modify permissions
USER root

# Grant Group 0 ownership and mirror User permissions to the Group
RUN chgrp -R 0 /opt/thunder/repository/resources/security && \
    chmod -R g=u /opt/thunder/repository/resources/security

# Switch back to the default non-root user
USER 1001
