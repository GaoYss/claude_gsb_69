# ---------- 构建阶段 ----------
FROM golang:1.27-alpine AS builder

# GOTOOLCHAIN=local 锁定镜像自带的 Go, 避免构建时联网下载工具链
ENV GOTOOLCHAIN=local \
    CGO_ENABLED=0

WORKDIR /src

# 先复制依赖清单, 依赖没变时可以直接命中构建缓存
COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://goproxy.cn,direct GOSUMDB=off && go mod download

COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# ---------- 运行阶段 ----------
FROM alpine:3.20

ENV TZ=Asia/Shanghai
RUN apk add --no-cache ca-certificates tzdata \
    && ln -snf /usr/share/zoneinfo/$TZ /etc/localtime \
    && echo "$TZ" > /etc/timezone

WORKDIR /app
COPY --from=builder /out/server /app/server
# sqlite 模式下数据文件落在 /app/data, docker-compose 会挂载数据卷持久化
RUN mkdir -p /app/data

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=15s --retries=5 \
    CMD wget -qO- http://127.0.0.1:8080/readyz || exit 1

CMD ["/app/server"]
