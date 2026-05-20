# ========== Stage 1: Build ==========
FROM golang:1.25-alpine AS builder

WORKDIR /build

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct

RUN apk --no-cache add gcc musl-dev git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /build/devops .

# ========== Stage 2: Runtime ==========
FROM alpine:3.23

WORKDIR /app

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

RUN apk --no-cache add \
    ca-certificates \
    openssh-client \
    tzdata \
    curl \
    bash \
    wget && \
    rm -rf /var/cache/apk/*

ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

RUN addgroup -S devops && adduser -S devops -G devops
RUN mkdir -p /app/log /app/upload /app/docs /data/inspection && chown -R devops:devops /app /data

COPY --from=builder /build/devops /app/devops
RUN chmod +x /app/devops

COPY --from=builder /build/common/templates /app/common/templates
RUN chown -R devops:devops /app

EXPOSE 8000

USER devops

# Runtime config substitution from env vars
CMD ["/bin/sh", "-c", "\
  cp /app/config.yaml /app/config.tmp.yaml && \
  sed \
    -e \"s|__DB_HOST__|${DB_HOST:-postgres}|g\" \
    -e \"s|__DB_PORT__|${DB_PORT:-5432}|g\" \
    -e \"s|__DB_NAME__|${DB_NAME:-autoops}|g\" \
    -e \"s|__DB_USER__|${DB_USER:-devops}|g\" \
    -e \"s|__DB_PASSWORD__|${DB_PASSWORD:-devops@2025}|g\" \
    -e \"s|__REDIS_ADDR__|${REDIS_ADDR:-valkey:6379}|g\" \
    -e \"s|__REDIS_PASSWORD__|${REDIS_PASSWORD:-zhangfan@123}|g\" \
    -e \"s|__IMAGE_HOST__|${IMAGE_HOST:-http://localhost:8080}|g\" \
    -e \"s|__GITOPS_REPO_PATH__|${GITOPS_REPO_PATH:-/workspace/pukka-gitops}|g\" \
    /app/config.tmp.yaml > /app/config.runtime.yaml && \
  exec /app/devops -c /app/config.runtime.yaml \
"]
