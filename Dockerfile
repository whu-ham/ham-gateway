# Dockerfile for HAM Gateway
FROM golang:1.26.1-alpine3.22 AS builder

WORKDIR /app

# Install protoc and dependencies
RUN apk add --no-cache protobuf git

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

RUN apk add --no-cache ca-certificates

WORKDIR /root/

COPY --from=builder /app/gateway .

EXPOSE 8080

CMD ["./gateway"]
