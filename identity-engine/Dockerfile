FROM node:24-alpine AS build
WORKDIR /app
RUN corepack enable && corepack prepare pnpm@9.15.9 --activate
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY . .
RUN pnpm build && pnpm test && pnpm test:engine
FROM node:24-alpine
WORKDIR /app
COPY --from=build /app/server.cjs /app/engine.cjs /app/verify.cjs ./
COPY --from=build /app/dist ./dist
COPY --from=build /app/vendor/lm-detector ./vendor/lm-detector
USER node
EXPOSE 8081
CMD ["node", "server.cjs"]
