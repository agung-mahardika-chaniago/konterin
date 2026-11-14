# syntax=docker/dockerfile:1.7
# FixPhone frontend image: Vite build served by an unprivileged Nginx that also
# reverse-proxies the API. Build from the repository root:
#   docker build -f deploy/docker/frontend.Dockerfile -t fixphone-frontend .
ARG NODE_VERSION=22

FROM node:${NODE_VERSION}-alpine AS build
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ ./

# Relative API path keeps the same origin (CORS-free) behind Nginx/Caddy.
ARG VITE_API_BASE_URL=/api/v1
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
RUN npm run build

FROM nginxinc/nginx-unprivileged:1.27-alpine AS runtime
USER 101
COPY deploy/nginx/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1

# The unprivileged image already runs nginx as uid 101.
