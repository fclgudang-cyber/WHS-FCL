FROM node:20-slim AS builder

# --max-old-space-size
ENV PNPM_HOME="/pnpm"
ENV PATH="$PNPM_HOME:$PATH"
ENV NODE_OPTIONS=--max-old-space-size=8192
ENV TZ=Asia/Shanghai

RUN corepack enable

# 可选：覆盖 .env.production 中的接口地址（Vite 中真实环境变量优先级高于 .env 文件）
ARG VITE_GLOB_API_URL=
ARG VITE_GLOB_SSE_URL=
ARG VITE_FCL_URL=

WORKDIR /app

# copy package.json and pnpm-lock.yaml to workspace
COPY . /app

RUN --mount=type=cache,id=pnpm,target=/pnpm/store pnpm install --frozen-lockfile
RUN if [ -n "$VITE_GLOB_API_URL" ]; then export VITE_GLOB_API_URL; fi && \
    if [ -n "$VITE_GLOB_SSE_URL" ]; then export VITE_GLOB_SSE_URL; fi && \
    if [ -n "$VITE_FCL_URL" ]; then export VITE_FCL_URL; fi && \
    pnpm run build --filter=@vben/web-antd

RUN echo "Builder Success 🎉"

FROM nginx:stable-alpine AS production

RUN echo "types { application/javascript js mjs; }" > /etc/nginx/conf.d/mjs.conf
COPY --from=builder /app/apps/admin/dist /usr/share/nginx/html

COPY --from=builder /app/scripts/deploy/nginx.conf /etc/nginx/nginx.conf

EXPOSE 8080

# start nginx
CMD ["nginx", "-g", "daemon off;"]
