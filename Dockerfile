# Stage 1: Build the Go backend
FROM golang:1.22-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git make

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=latest" \
    -o /build/axiom-server \
    cmd/axiom-server/main.go

# Stage 2: Build frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

# Stage 3: Runtime - nginx serving frontend with env vars
FROM nginx:alpine

RUN apk add --no-cache curl && rm -rf /var/cache/apk/*

# Copy nginx config
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Copy Go binary and create wrapper
COPY --from=builder /build/axiom-server /app/axiom-server
RUN mkdir -p /app/data && chmod +x /app/axiom-server

# Create health check script
RUN echo '#!/bin/sh
curl -sf http://localhost/health || exit 1' > /usr/local/bin/healthcheck && chmod +x /usr/local/bin/healthcheck

# Copy frontend build
COPY --from=frontend-builder /app/dist /usr/share/nginx/html

# Set working directory
WORKDIR /app

# Expose port
EXPOSE 80

# Run backend in background, then nginx
CMD sh -c 'axiom-server & sleep 2; exec nginx -g "daemon off;"'
