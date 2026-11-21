# Контекст сборки — корень репозитория (см. deploy/compose/docker-compose.yml).
ARG GO_VERSION=1.27.1

# ---------- build ----------
FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /app

# Сначала только манифесты модулей — слой с зависимостями кэшируется,
# пока go.mod/go.sum не менялись.
COPY go.work go.work.sum ./
COPY platform/go.mod platform/go.sum ./platform/
COPY shared/go.mod ./shared/
COPY trader/go.mod trader/go.sum ./trader/

RUN go mod download

COPY platform ./platform
COPY shared ./shared
COPY trader ./trader

# GOARCH не фиксируем: на Apple Silicon соберётся arm64, на Windows/Intel — amd64.
ARG TARGETOS=linux
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /app/app-trader ./trader/cmd/main.go

# ---------- runtime ----------
FROM alpine:3.21

# ca-certificates — для HTTPS к Telegram Bot API, tzdata — для корректного времени в логах.
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S appgroup && adduser -S appuser -G appgroup \
    && mkdir /data && chown appuser:appgroup /data

WORKDIR /app

COPY --from=builder /app/app-trader .

USER appuser

# Состояние стоп-крана (KILLSWITCH_STATE_FILE) лежит в /data — это том в compose.
VOLUME ["/data"]

ENTRYPOINT ["./app-trader"]
