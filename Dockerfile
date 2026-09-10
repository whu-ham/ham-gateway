# Dockerfile for HAM Gateway
FROM golang:1.26.1-alpine3.22 AS builder

# Allow overriding the Go module proxy for builds behind networks where
# proxy.golang.org is unreachable (e.g. pass --build-arg GOPROXY=https://goproxy.cn,direct).
# Defaults to the standard proxy so CI behavior is unchanged.
ARG GOPROXY=https://proxy.golang.org,direct
# Optional Alpine CDN mirror host (e.g. mirrors.aliyun.com) for slow/blocked
# default CDN. Empty default keeps CI behavior unchanged.
ARG ALPINE_MIRROR=
ENV GOPROXY=${GOPROXY}

WORKDIR /app

# Install protoc and dependencies
RUN if [ -n "$ALPINE_MIRROR" ]; then sed -i "s|dl-cdn.alpinelinux.org|$ALPINE_MIRROR|g" /etc/apk/repositories; fi && \
    apk add --no-cache protobuf git

# Copy go.mod and go.sum
COPY go.mod go.sum ./
RUN go mod download

# Install protoc-gen-go and protoc-gen-go-grpc
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Copy source code
COPY . .

# Create gen directory and generate protobuf code using go generate
RUN mkdir -p gen && go generate ./...

# Build the application
RUN go build -o /app/gateway cmd/gateway/main.go

# Final stage
FROM alpine:latest

ARG ALPINE_MIRROR=
RUN if [ -n "$ALPINE_MIRROR" ]; then sed -i "s|dl-cdn.alpinelinux.org|$ALPINE_MIRROR|g" /etc/apk/repositories; fi && \
    apk add --no-cache ca-certificates

WORKDIR /root/

COPY --from=builder /app/gateway .

EXPOSE 8080

CMD ["./gateway"]
