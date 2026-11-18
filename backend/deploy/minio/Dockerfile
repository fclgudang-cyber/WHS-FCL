# MinIO 社区版镜像已从 Docker Hub / quay.io 下架（2025-09 后官方只发 GitHub release 二进制）。
# 这里直接从官方 GitHub release 下载二进制并做 sha256 校验，构建本地镜像。
FROM docker.io/library/alpine:3.22

ARG TARGETARCH
ARG MINIO_VERSION=RELEASE.2025-09-07T16-13-09Z

RUN apk --no-cache add ca-certificates curl && \
    case "${TARGETARCH:-amd64}" in \
      amd64) arch=linux-amd64 ;; \
      arm64) arch=linux-arm64 ;; \
      *) echo "unsupported arch: ${TARGETARCH}" >&2; exit 1 ;; \
    esac && \
    bin="minio.${MINIO_VERSION}" && \
    curl -fSL --retry 3 -o "/tmp/${bin}" "https://github.com/minio/minio/releases/download/${MINIO_VERSION}/minio.${arch}.${MINIO_VERSION}" && \
    curl -fSL --retry 3 -o "/tmp/${bin}.sha256sum" "https://github.com/minio/minio/releases/download/${MINIO_VERSION}/minio.${arch}.${MINIO_VERSION}.sha256sum" && \
    cd /tmp && sha256sum -c "${bin}.sha256sum" && \
    install -m 0755 "/tmp/${bin}" /usr/local/bin/minio && \
    rm -f "/tmp/${bin}" "/tmp/${bin}.sha256sum"

EXPOSE 9000 9001
VOLUME /data
ENTRYPOINT ["/usr/local/bin/minio"]
CMD ["server", "/data", "--console-address", ":9001"]
