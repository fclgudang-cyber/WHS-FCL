#!/bin/sh
# 在容器启动前把 /app/configs/*.yaml 里的 ${VAR:default} 占位符渲染成真实值。
# Kratos 自带的 resolver 只从配置文件里取占位符的值，不会读环境变量，
# 所以这里用 sed 做一次替换；没有设置的环境变量会使用与旧静态配置完全相同的默认值。
set -eu

render_var() {
  name=$1
  default=$2

  eval "value=\${$name-}"
  if [ -z "$value" ]; then
    value=$default
  fi

  # 转义 sed 替换串里的特殊字符（\ & |）
  escaped=$(printf '%s' "$value" | sed -e 's/[\\&|]/\\&/g')

  # '$' 和 '{' 必须让 sed 当字面量处理；拆开拼接避免被 shell 当成变量展开
  a='\$'
  b='{'
  c=':[^}]*}'
  pattern="$a$b$name$c"

  for f in /app/configs/*.yaml; do
    [ -f "$f" ] || continue
    sed -i "s|$pattern|$escaped|g" "$f"
  done
}

render_var DB_HOST postgres
render_var DB_PORT 5432
render_var DB_USER postgres
render_var DB_PASSWORD '*Abcd123456'
render_var DB_NAME ghost_wms
render_var REDIS_ADDR redis:6379
render_var REDIS_PASSWORD '*Abcd123456'
render_var ETCD_ENDPOINT etcd:2379
render_var MINIO_ENDPOINT minio:9000
render_var MINIO_ACCESS_KEY root
render_var MINIO_SECRET_KEY '*Abcd123456'
render_var MINIO_UPLOAD_HOST 127.0.0.1:9000
render_var MINIO_DOWNLOAD_HOST http://127.0.0.1:9000
render_var TRACE_ENDPOINT jaeger:4317
render_var jwt_signing_key dev_only_change_me_in_prod
render_var aes_key f51d66a73d8a0927

exec "$@"
