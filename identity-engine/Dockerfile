FROM node:24.22.1-alpine AS build
RUN apk add --no-cache 'ca-certificates>=20261020-r0'
WORKDIR /app
RUN corepack enable && corepack prepare pnpm@9.15.9 --activate
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY . .
RUN pnpm build && pnpm test && pnpm test:engine
FROM node:24.22.1-alpine
RUN apk upgrade --no-cache && apk add --no-cache 'ca-certificates>=20261020-r0' \
    && rm -rf /usr/local/lib/node_modules/npm /usr/local/lib/node_modules/corepack \
    && rm -f /usr/local/bin/npm /usr/local/bin/npx /usr/local/bin/corepack \
       /usr/local/bin/pnpm /usr/local/bin/pnpx /usr/local/bin/yarn /usr/local/bin/yarnpkg
WORKDIR /app
COPY --from=build /app/server.cjs /app/engine.cjs /app/verify.cjs ./
COPY --from=build /app/dist ./dist
COPY --from=build /app/vendor/lm-detector ./vendor/lm-detector
USER node
EXPOSE 8081
CMD ["node", "server.cjs"]
