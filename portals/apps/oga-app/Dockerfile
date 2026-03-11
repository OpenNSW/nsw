FROM node:22-bookworm-slim AS builder

WORKDIR /workspace

RUN corepack enable

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml tsconfig.json ./
COPY apps/trader-app/package.json apps/trader-app/package.json
COPY apps/oga-app/package.json apps/oga-app/package.json
COPY packages/jsonforms-renderers/package.json packages/jsonforms-renderers/package.json
COPY packages/ui/package.json packages/ui/package.json

RUN pnpm install --frozen-lockfile

COPY . .

# Build oga-app and any workspace dependencies it relies on (e.g. local packages/*).
RUN pnpm --filter oga-app... build

FROM nginx:1.27-alpine

LABEL org.opencontainers.image.source="https://github.com/OpenNSW/nsw"
LABEL org.opencontainers.image.description="NSW OGA Officer Portal Frontend"

COPY apps/oga-app/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /workspace/apps/oga-app/dist /usr/share/nginx/html
COPY apps/oga-app/docker-entrypoint.sh /docker-entrypoint.d/40-runtime-env.sh

RUN chmod +x /docker-entrypoint.d/40-runtime-env.sh

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:80/ || exit 1

CMD ["nginx", "-g", "daemon off;"]
