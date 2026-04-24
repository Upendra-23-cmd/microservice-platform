# ============================================================
# Stage 1: Dependency cache
# ============================================================
FROM node:20-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --frozen-lockfile

# ============================================================
# Stage 2: Builder
# ============================================================
FROM node:20-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .

ARG VITE_API_BASE_URL=/api/v1
ARG VITE_APP_VERSION=1.0.0
ARG VITE_APP_ENV=production
ARG VITE_ENABLE_DEVTOOLS=false

ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
ENV VITE_APP_VERSION=$VITE_APP_VERSION
ENV VITE_APP_ENV=$VITE_APP_ENV
ENV VITE_ENABLE_DEVTOOLS=$VITE_ENABLE_DEVTOOLS

RUN npm run build

# ============================================================
# Stage 3: Production nginx
# ============================================================
FROM nginx:1.25-alpine AS production

# Remove default nginx config
RUN rm -rf /usr/share/nginx/html/* /etc/nginx/conf.d/default.conf

# Copy custom nginx config
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Copy built assets
COPY --from=builder /app/dist /usr/share/nginx/html

# Run as non-root
RUN addgroup -S appgroup && adduser -S appuser -G appgroup && \
    chown -R appuser:appgroup /usr/share/nginx/html /var/cache/nginx /var/log/nginx /etc/nginx/conf.d && \
    touch /var/run/nginx.pid && chown appuser:appgroup /var/run/nginx.pid

USER appuser
EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
