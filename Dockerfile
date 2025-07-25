# Build stage
FROM golang:1.24-alpine AS builder

# Install git (required for go mod download)
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy app source code
COPY . .

# Build the application from correct path
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o tempfile ./cmd/server

# Final stage
FROM alpine:latest

# Install ca-certificates and wget for HTTPS requests and healthcheck
RUN apk --no-cache add ca-certificates wget

# Create non-root user
RUN addgroup -g 1001 -S tempfile && \
    adduser -S tempfile -u 1001 -G tempfile

# Set working directory
WORKDIR /app

# Create uploads directory with proper permissions
# Note: We no longer need web/static and web/templates as they're embedded
RUN mkdir -p uploads && \
    chown -R tempfile:tempfile /app

# Copy binary from builder stage (now contains embedded assets)
COPY --from=builder --chown=tempfile:tempfile /app/tempfile .

# Switch to non-root user
USER tempfile

# Expose port
EXPOSE 3000

# Health check using dedicated health endpoint
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3000/health || exit 1

# Set default environment variables
ENV PORT=3000
ENV APP_ENV=production
ENV DEBUG=false
ENV PUBLIC_URL=http://localhost:3000
ENV ENABLE_WEB_UI=true

# Run the application (now fully standalone with embedded assets)
CMD ["./tempfile"]
