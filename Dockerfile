# Central-Devtron — single-container image: Go API + embedded React frontend,
# backed by embedded SQLite (data on a mounted volume at /data).

# ── Stage 1: build the frontend ──────────────────────────────────────────────
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# ── Stage 2: build the backend, embedding the frontend build ─────────────────
FROM golang:1.25-alpine AS backend
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# Replace the placeholder dist with the real build so go:embed ships the UI.
COPY --from=frontend /app/frontend/dist ./internal/web/dist
# Pure-Go SQLite (modernc) → CGO can stay off, producing a static binary.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

# ── Stage 3: minimal runtime ─────────────────────────────────────────────────
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 65532 appuser \
    && mkdir -p /data && chown appuser:appuser /data
COPY --from=backend /server /usr/local/bin/server
# /data holds the SQLite file; mount a volume/PVC here so data survives restarts.
ENV DB_PATH=/data/central.db \
    PORT=8080
EXPOSE 8080
USER appuser
VOLUME ["/data"]
ENTRYPOINT ["server"]
