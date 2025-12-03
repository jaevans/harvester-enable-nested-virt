# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /workspace

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ cmd/
COPY pkg/ pkg/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o webhook ./cmd/webhook

# Final stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /workspace/webhook .

# Run as non-root user
RUN addgroup -g 1000 webhook && \
    adduser -D -u 1000 -G webhook webhook && \
    chown -R webhook:webhook /app

USER webhook

ENTRYPOINT ["/app/webhook"]
