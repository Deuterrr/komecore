# --- Build --- #
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -mod=readonly -ldflags="-w -s" \
    -o /out/komecore \
    ./cmd/komecore/main.go

# --- Runtime --- #
FROM alpine:3.21 AS runtime

LABEL org.opencontainers.image.title="komecore"
LABEL org.opencontainers.image.description="High-performance general e-commerce backend service in Go"
LABEL org.opencontainers.image.authors="Deuterrr"

RUN apk add --no-cache wget && \
    addgroup -S appgroup && \
    adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /out/komecore  ./komecore
COPY --from=builder /app/migrations ./migrations

EXPOSE 7129

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
  CMD wget -qO- http://localhost:7129/health || exit 1

USER appuser

ENTRYPOINT ["/app/komecore"]
