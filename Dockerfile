# syntax=docker/dockerfile:1
# match go.mod
FROM golang:1.25-alpine AS builder  

# Install build deps only
RUN apk add --no-cache ca-certificates git tzdata && update-ca-certificates

WORKDIR /build

# Copy dependencies first for caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source and build ONE binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ticketsphere-server .

# Final stage: minimal Alpine
FROM alpine:3.19

# Runtime essentials only
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 1001 appuser

WORKDIR /app

COPY --from=builder /build/ticketsphere-server .
# Copy migrations for auto-run (optional)

USER appuser

EXPOSE 8080 8081

HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

ENTRYPOINT ["./ticketsphere-server"]