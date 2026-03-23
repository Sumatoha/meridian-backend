FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata \
    # Chromium + deps for headless browser scraping (Rod)
    chromium nss freetype harfbuzz font-noto-cjk

# Tell Rod to use system chromium instead of downloading its own
ENV ROD_BROWSER=/usr/bin/chromium-browser

WORKDIR /app
COPY --from=builder /server .
COPY migrations ./migrations

EXPOSE 8080

CMD ["./server"]
