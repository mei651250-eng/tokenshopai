# 多阶段构建
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
# 限制堆内存 + 关闭压缩，避免小内存容器构建时 OOM 被 Killed
ENV NODE_OPTIONS="--max-old-space-size=4096"
RUN npm run build -- --minify false

# 运行阶段 - 使用 nginx 提供 SPA 服务
FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
