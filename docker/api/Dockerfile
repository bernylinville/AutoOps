# ========== Stage 1: Build ==========
FROM golang:1.25-alpine AS builder

WORKDIR /build

# 设置 Go 国内代理
ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct

# 安装编译依赖
RUN apk --no-cache add gcc musl-dev git

# 先复制依赖文件，利用 Docker 缓存
COPY go.mod go.sum ./
RUN go mod download

# 复制源码并编译
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /build/devops .

# ========== Stage 2: Runtime ==========
FROM alpine:3.23

WORKDIR /app

# 设置 Alpine 国内镜像源
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

# 安装运行时依赖（SSH用于远程管理主机）
RUN apk --no-cache add \
    ca-certificates \
    openssh-client \
    tzdata \
    curl \
    bash \
    git \
    gcc \
    musl-dev && \
    rm -rf /var/cache/apk/*

# 设置时区
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 配置 Go 环境变量
ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    GOPATH=/go \
    GOCACHE=/go-cache

# 创建必要的目录
RUN mkdir -p /app/log /app/upload /app/docs /go/pkg/mod /go-cache && \
    chmod -R 777 /go /go-cache

# 从构建阶段复制二进制
COPY --from=builder /build/devops /app/devops
RUN chmod +x /app/devops

# 复制运行时需要的资源文件（工具列表/服务模版等）
COPY --from=builder /build/common/templates /app/common/templates

EXPOSE 8000

CMD ["/app/devops"]
