# ========== Stage 1: Build ==========
FROM node:20-alpine AS builder

WORKDIR /app

# 设置 npm 国内镜像
RUN npm config set registry https://registry.npmmirror.com

# 先复制依赖文件，利用 Docker 缓存
COPY web/package.json web/package-lock.json ./
RUN npm ci

# 复制源码并构建
COPY web/ ./
RUN npm run build

# ========== Stage 2: Serve ==========
FROM nginx:1.28.2-alpine

# 删除默认配置和静态文件
RUN rm -rf /usr/share/nginx/html/*

# 从构建阶段复制 dist
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
