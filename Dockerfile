# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.22
ARG NODE_VERSION=20

FROM golang:${GO_VERSION}-alpine AS backend-builder
WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG GIT_SHA=unknown
ENV CGO_ENABLED=0

RUN go build -trimpath -ldflags="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" -o /out/axiom-server ./cmd/axiom-server

FROM node:${NODE_VERSION}-alpine AS frontend-builder
WORKDIR /src/web

COPY web/package.json web/package-lock.json ./
RUN npm ci --ignore-scripts

COPY web/ ./
RUN find src -type f \( -name '*.js' -o -name '*.js.map' -o -name '*.d.ts' -o -name '*.d.ts.map' \) -delete \
    && npm run build

FROM nginxinc/nginx-unprivileged:1.27-alpine AS runtime

USER root

RUN apk add --no-cache bash curl ca-certificates tzdata \
    && mkdir -p /tmp/axiom /var/cache/nginx /var/run /var/lib/axiom \
    && chown -R 101:101 /tmp/axiom /var/cache/nginx /var/run /var/lib/axiom

COPY --from=backend-builder /out/axiom-server /usr/local/bin/axiom-server
COPY --from=frontend-builder /src/web/dist /usr/share/nginx/html
COPY scripts/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

RUN cat <<'EOF' >/etc/nginx/conf.d/default.conf
server {
    listen 8080;
    server_name _;
    server_tokens off;

    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "DENY" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    add_header Permissions-Policy "geolocation=(), microphone=(), camera=()" always;
    add_header Content-Security-Policy "default-src 'self'; img-src 'self' data: https:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self' http://127.0.0.1:8081;" always;

    location /api/ {
        proxy_pass http://127.0.0.1:8081;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /health {
        proxy_pass http://127.0.0.1:8081;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
    }

    location / {
        root /usr/share/nginx/html;
        index index.html;
        try_files $uri $uri/ /index.html;
    }

    location /assets/ {
        root /usr/share/nginx/html;
        expires 30d;
        add_header Cache-Control "public, immutable" always;
    }
}
EOF

RUN chmod +x /usr/local/bin/axiom-server /usr/local/bin/docker-entrypoint.sh \
    && chown -R 101:101 /usr/share/nginx/html /etc/nginx/conf.d /usr/local/bin/axiom-server /usr/local/bin/docker-entrypoint.sh

ENV AXIOM_HOST=0.0.0.0 \
    AXIOM_PORT=8081 \
    AXIOM_ENV=production \
    AXIOM_LOG_LEVEL=info \
    AXIOM_DB_DRIVER=sqlite3 \
    AXIOM_DB_URL=file:/var/lib/axiom/axiom.db \
    AXIOM_SESSION_SECRET=change-me-in-production \
    AXIOM_AI_BACKEND=local \
    AXIOM_AI_TIMEOUT=90s \
    AXIOM_AI_MAX_TOKENS=768

USER 101

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD curl -fsS http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
