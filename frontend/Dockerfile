# ---------- 构建阶段 ----------
FROM node:22-alpine AS builder

# 与后端 GOPROXY 保持一致, 默认走国内镜像; 需要时可覆盖:
#   docker build --build-arg NPM_REGISTRY=https://registry.npmjs.org/ .
ARG NPM_REGISTRY=https://registry.npmmirror.com

WORKDIR /app

# 先复制依赖清单, 依赖没变时可以直接命中构建缓存
COPY package.json package-lock.json ./
RUN npm config set registry "$NPM_REGISTRY" && npm ci --no-audit --no-fund

COPY . .
RUN npm run build

# ---------- 运行阶段 ----------
FROM nginx:1.27-alpine

ENV TZ=Asia/Shanghai
RUN apk add --no-cache tzdata \
    && ln -snf /usr/share/zoneinfo/$TZ /etc/localtime \
    && echo "$TZ" > /etc/timezone

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=5 \
    CMD wget -qO- http://127.0.0.1/ >/dev/null || exit 1

CMD ["nginx", "-g", "daemon off;"]
