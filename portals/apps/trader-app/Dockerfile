FROM node:22-alpine AS build
RUN corepack enable && corepack prepare pnpm@10.28.1 --activate
WORKDIR /workspace
COPY package.json pnpm-workspace.yaml pnpm-lock.yaml ./
COPY packages/ui/package.json packages/ui/
COPY packages/jsonforms-renderers/package.json packages/jsonforms-renderers/
COPY apps/trader-app/package.json apps/trader-app/
RUN pnpm install --frozen-lockfile
COPY packages/ packages/
COPY apps/trader-app/ apps/trader-app/
RUN pnpm --filter trader-app build

FROM nginx:1.27-alpine
COPY --from=build /workspace/apps/trader-app/dist /usr/share/nginx/html
COPY apps/trader-app/nginx.conf /etc/nginx/conf.d/default.conf
COPY apps/trader-app/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
EXPOSE 80
ENTRYPOINT ["/entrypoint.sh"]
