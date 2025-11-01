# ---- Build stage ----
FROM golang:1.24.5-alpine AS builder

WORKDIR /app

# Install libvips + build tools
RUN apk add --no-cache build-base vips-dev git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ENV CGO_ENABLED=1

RUN go build -ldflags="-s -w" -o photo-backend ./cmd/api

# ---- Runtime stage ----
FROM alpine:3.19

WORKDIR /app

# Install libvips runtime + other dependencies
RUN apk add --no-cache vips ca-certificates tzdata bash tini

# Copy the binary and config
COPY --from=builder /app/photo-backend .
COPY config/docker.yaml ./config.yaml

EXPOSE 8080

# Use tini as init process
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["./photo-backend", "--config=config.yaml"]
