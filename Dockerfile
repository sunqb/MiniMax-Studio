# ===== Build Stage =====
FROM golang:1.25-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git ca-certificates

# 先缓存依赖层
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码并编译
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /build/minimax-studio \
    ./cmd/server

# ===== Runtime Stage =====
FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget tzdata su-exec && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app
RUN mkdir -p /app/data
COPY --from=builder /build/minimax-studio /app/minimax-studio
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["/app/docker-entrypoint.sh"]
