# syntax=docker/dockerfile:1
FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

FROM node:22-alpine AS builder
WORKDIR /app
# Rewrites in next.config.ts are resolved at build time, so the API address is a build arg.
# It is also set at runtime below, where server components fetch it directly.
ARG API_URL=http://api:8080
ENV API_URL=$API_URL NEXT_TELEMETRY_DISABLED=1
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM node:22-alpine
WORKDIR /app
ARG API_URL=http://api:8080
ENV NODE_ENV=production NEXT_TELEMETRY_DISABLED=1 PORT=3000 HOSTNAME=0.0.0.0 API_URL=$API_URL
RUN adduser -D -u 10001 app
COPY --from=builder --chown=app /app/.next/standalone ./
COPY --from=builder --chown=app /app/.next/static ./.next/static
USER app
EXPOSE 3000
CMD ["node", "server.js"]
