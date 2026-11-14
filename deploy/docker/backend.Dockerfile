# syntax=docker/dockerfile:1.7
# FixPhone backend image: API, worker, and CLI in one multi-stage build.
# Build from the repository root:
#   docker build -f deploy/docker/backend.Dockerfile -t fixphone-backend .
ARG GO_VERSION=1.26.11

FROM golang:${GO_VERSION}-alpine AS build
RUN apk add --no-cache ca-certificates tzdata git
ENV CGO_ENABLED=0 GOOS=linux
WORKDIR /src

# Cache module downloads separately from source changes.
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

ARG VERSION=1.0.0
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
RUN set -eux; \
    ldflags="-s -w \
      -X fixphone/backend/internal/pkg/version.Version=${VERSION} \
      -X fixphone/backend/internal/pkg/version.Commit=${COMMIT} \
      -X fixphone/backend/internal/pkg/version.BuildDate=${BUILD_DATE}"; \
    go build -trimpath -ldflags "$ldflags" -o /out/fixphone-api ./cmd/api; \
    go build -trimpath -ldflags "$ldflags" -o /out/fixphone-worker ./cmd/worker; \
    go build -trimpath -ldflags "$ldflags" -o /out/fixphone-cli ./cmd/cli

# Runtime image: minimal Alpine, non-root uid 10001, no shell tooling beyond
# busybox wget for the container health check.
FROM alpine:3.21 AS runtime
RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -g 10001 -S fixphone \
    && adduser -u 10001 -S -G fixphone -h /app fixphone \
    && mkdir -p /app/uploads /app/backups \
    && chown -R fixphone:fixphone /app

COPY --from=build /out/ /usr/local/bin/

WORKDIR /app
USER 10001:10001
EXPOSE 8080

# /healthz is liveness only; orchestration should prefer /readyz.
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1

ENTRYPOINT ["/usr/local/bin/fixphone-api"]
