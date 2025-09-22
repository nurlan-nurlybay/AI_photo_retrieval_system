# ---- Build stage ----
FROM golang:1.24.5-alpine AS builder

WORKDIR /app

# cache de
COPY go.mod go.sum ./
RUN go mod download

# copy sour
COPY . .

# build binary (stripped)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o photo-backend ./cmd/server

# ---- Runtime stage ----
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/photo-backend .
COPY config/docker.yaml ./config.yaml

# tini makes signals cleaner
RUN apk add --no-cache ca-certificates tzdata bash tini

EXPOSE 8080

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["./photo-backend", "--config=config.yaml"]
