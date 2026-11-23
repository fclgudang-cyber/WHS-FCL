##################################
# 第一阶段：构建GO可执行文件
##################################

# 使用官方的 Go 基础镜像作为构建环境（与 go.mod 的 go 版本对齐，避免 GOTOOLCHAIN 联网下载）
ARG GO_VERSION=1.26.4
FROM golang:${GO_VERSION}-alpine AS builder

ARG SERVICE_NAME=app
ARG APP_VERSION=1.0.0

# 设置工作目录
WORKDIR /src

# 复制项目源代码到工作目录
COPY . /src

# Unduh dependensi memakai proxy Go resmi.
# GOSUMDB tidak dimatikan: checksum tetap diverifikasi terhadap sum.golang.org
# (gemakan GOSUMDB=off sebelumnya hanya menghindari mirror lama, bukan kebutuhan proyek).
RUN GOPROXY=https://proxy.golang.org,direct go mod download

# 编译可执行文件（使用WORKDIR和相对路径，而不是cd）
RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build -ldflags "-X main.version=$APP_VERSION" \
    -o /src/bin/${SERVICE_NAME}-server ./app/${SERVICE_NAME}/service/cmd/server/

# 复制配置文件到统一目录
# Config deploy juga ditanamkan ke image agar tidak bergantung pada bind mount host
# (Coolify sering tidak menyalin file YAML tersebut ke direktori aplikasi).
RUN mkdir -p /src/bin/configs && \
    if [ -d "/src/app/${SERVICE_NAME}/service/configs" ]; then \
      cp -r /src/app/${SERVICE_NAME}/service/configs/* /src/bin/configs/ 2>/dev/null || true; \
    fi && \
    if [ -f "/src/deploy/compose/${SERVICE_NAME}-data.yaml" ]; then \
      cp "/src/deploy/compose/${SERVICE_NAME}-data.yaml" /src/bin/configs/data.yaml; \
    fi && \
    for f in registry oss trace; do \
      if [ -f "/src/deploy/compose/${f}.yaml" ]; then \
        cp "/src/deploy/compose/${f}.yaml" "/src/bin/configs/${f}.yaml"; \
      fi; \
    done

##################################
# 第二阶段：创建最终的运行时镜像
##################################

# 使用 Alpine 作为基础镜像，因为它非常轻量级
FROM docker.io/library/alpine:3.22

ARG SERVICE_NAME=app

# 安装必要的证书（如果应用程序需要进行 HTTPS 请求）
RUN apk --no-cache add ca-certificates libcap

# 设置工作目录
WORKDIR /app

# 从第一阶段的构建结果中复制可执行文件到当前工作目录
COPY --from=builder /src/bin/${SERVICE_NAME}-server /app/bin/server

# 拷贝配置文件
COPY --from=builder /src/bin/configs/ /app/configs/

# 启动前根据环境变量渲染配置占位符（Kratos 自带 resolver 不读环境变量）
COPY deploy/entrypoint.sh /app/bin/entrypoint.sh

# 创建一个名为 appuser 的非 root 用户，并授予渲染配置所需的写权限
RUN setcap cap_net_bind_service=+ep /app/bin/server && \
    adduser -D appuser && \
    mkdir -p /app/uploads && \
    chmod +x /app/bin/entrypoint.sh && \
    chown -R appuser:appuser /app/bin /app/configs /app/uploads

# 切换到非特权用户
USER appuser:appuser

# 暴露服务端口，根据你的实际服务端口进行修改

# 设置容器启动时执行的命令
ENTRYPOINT ["/app/bin/entrypoint.sh"]
CMD ["/app/bin/server", "-c", "/app/configs"]
