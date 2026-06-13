# Stage 1: Build
FROM docker.1ms.run/library/golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Go module proxy for Chinese mainland users.
ENV GOPROXY=https://goproxy.cn,direct

# Cache module downloads in a separate layer.
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source and build.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o /build/iptv-builder \
    ./cmd/iptv-builder

# Stage 2: Runtime
FROM docker.1ms.run/library/alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/iptv-builder .

# Default mount points for configuration and output.
VOLUME ["/config", "/output", "/cache"]

ENTRYPOINT ["./iptv-builder"]
