# Stage 1: Build the Go binary
FROM golang:1.26-alpine AS builder

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /usr/local/bin/goog ./cmd/goog

# Stage 2: Runtime with Chrome
FROM alpine:3.21

# Install runtime dependencies
RUN apk add --no-cache \
    chromium \
    chromium-chromedriver \
    bash \
    jq \
    github-cli \
    # Fonts for rendering
    font-noto \
    font-noto-cjk \
    && rm -rf /var/cache/apk/*

# Set Chrome environment variables for chromedp
ENV CHROME_BIN=/usr/bin/chromium-browser
ENV CHROMEDP_NO_SANDBOX=true

# Copy the built binary
COPY --from=builder /usr/local/bin/goog /usr/local/bin/goog

# Copy the entrypoint script
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
