# MinIO community images were removed from Docker Hub / quay.io after 2025-09,
# so the official GitHub release binary is downloaded at build time.
# GitHub can be slow from Indonesia: long timeouts, retries and fallback mirrors
# are used, with the official sha256 pinned below for integrity.
FROM docker.io/library/alpine:3.22

ARG TARGETARCH
ARG MINIO_VERSION=RELEASE.2025-09-07T16-13-09Z
ARG MINIO_SHA_AMD64=7c5bd8512c6e966455b1d198209358b2d191c77a83ab377c4073281065fb855f
ARG MINIO_SHA_ARM64=5c83cd2cf151717ba0243f73e1c7802ff36e272b67144bdd7f1f7d684fd6f03d

RUN apk --no-cache add ca-certificates curl && \
    case "${TARGETARCH:-amd64}" in \
      amd64) arch=linux-amd64; expected="${MINIO_SHA_AMD64}" ;; \
      arm64) arch=linux-arm64; expected="${MINIO_SHA_ARM64}" ;; \
      *) echo "unsupported arch: ${TARGETARCH}" >&2; exit 1 ;; \
    esac && \
    bin="minio.${MINIO_VERSION}.${arch}" && \
    url="https://github.com/minio/minio/releases/download/${MINIO_VERSION}/minio.${arch}.${MINIO_VERSION}" && \
    for src in \
      "${url}" \
      "https://ghproxy.net/${url}" \
      "https://ghfast.top/${url}" \
      "https://gh-proxy.com/${url}" \
    ; do \
      echo "Downloading MinIO from ${src}" && \
      curl -fSL --retry 5 --retry-all-errors --retry-delay 3 \
        --connect-timeout 30 --max-time 900 -o "/tmp/${bin}" "${src}" && break; \
    done && \
    test -s "/tmp/${bin}" && \
    printf '%s  /tmp/%s\n' "${expected}" "${bin}" | sha256sum -c - && \
    install -m 0755 "/tmp/${bin}" /usr/local/bin/minio && \
    rm -f "/tmp/${bin}"

EXPOSE 9000 9001
VOLUME /data
ENTRYPOINT ["/usr/local/bin/minio"]
CMD ["server", "/data", "--console-address", ":9001"]
