# Единый образ Advisor: Go-бинарь advisord отдаёт и JSON API, и собранный SPA.
# Три стадии: сборка фронта (node) → сборка бинаря (go) → тонкий рантайм (alpine).

# --- 1. Фронтенд (SPA) ---
FROM node:20-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build   # → /web/dist

# --- 2. Бэкенд (Go, без CGo — modernc.org/sqlite чистый Go) ---
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/advisord ./cmd/advisord

# --- 3. Рантайм ---
FROM alpine:3.21 AS runtime
RUN apk add --no-cache ca-certificates tzdata wget && \
    adduser -D -H -u 10001 advisor
WORKDIR /app
COPY --from=build /out/advisord /app/advisord
COPY --from=web /web/dist /app/web
RUN mkdir -p /data && chown advisor /data

ENV ADVISOR_ADDR=:8080 \
    ADVISOR_DB=/data/server.db \
    ADVISOR_WEB=/app/web
EXPOSE 8080
VOLUME ["/data"]
USER advisor

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/api/health || exit 1

ENTRYPOINT ["/app/advisord"]
