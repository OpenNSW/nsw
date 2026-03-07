# Unified portals Dockerfile: build trader-app or oga-app with injectable env
FROM node:22-alpine AS builder

RUN corepack enable && corepack prepare pnpm@10.28.1 --activate
WORKDIR /app

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps ./apps
COPY packages ./packages
COPY tsconfig.json ./

# Build-time env (Vite embeds these)
ARG APP_NAME
ENV APP_NAME=${APP_NAME}

ARG VITE_IDP_BASE_URL=http://localhost:8090
ENV VITE_IDP_BASE_URL=${VITE_IDP_BASE_URL}

ARG VITE_API_BASE_URL=http://localhost:8080/api/v1
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}

ARG VITE_APP_URL=http://localhost:5173
ENV VITE_APP_URL=${VITE_APP_URL}

ARG VITE_IDP_CLIENT_ID=TRADER_PORTAL_APP
ENV VITE_IDP_CLIENT_ID=${VITE_IDP_CLIENT_ID}

# OGA-app specific (ignored for trader-app)
ARG VITE_OGA_API_BASE_URL=http://localhost:8081
ENV VITE_OGA_API_BASE_URL=${VITE_OGA_API_BASE_URL}

ARG VITE_INSTANCE_CONFIG=npqs
ENV VITE_INSTANCE_CONFIG=${VITE_INSTANCE_CONFIG}

ARG VITE_PORT=5174
ENV VITE_PORT=${VITE_PORT}

RUN pnpm install --frozen-lockfile
RUN pnpm --filter "${APP_NAME}" run build
RUN cp -r /app/apps/${APP_NAME}/dist /app/dist

# Serve with nginx
FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html

# SPA: serve index.html for all routes
RUN echo 'server { \
  root /usr/share/nginx/html; \
  index index.html; \
  location / { try_files $uri $uri/ /index.html; } \
}' > /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
