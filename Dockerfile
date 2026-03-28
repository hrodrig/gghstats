FROM golang:1.26.1-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILDDATE=unknown

RUN go build -ldflags "-s -w \
    -X 'github.com/hrodrig/gghstats/internal/version.Version=${VERSION}' \
    -X 'github.com/hrodrig/gghstats/internal/version.Commit=${COMMIT}' \
    -X 'github.com/hrodrig/gghstats/internal/version.BuildDate=${BUILDDATE}'" \
    -o gghstats ./cmd/gghstats

# Use Alpine 3.21 to avoid OpenSSL 3.5.x high CVEs present in 3.23.
FROM alpine:3.21
RUN apk add --no-cache ca-certificates && \
    addgroup -g 1000 gghstats && \
    adduser -D -u 1000 -G gghstats -h /data gghstats
COPY --from=builder /app/gghstats /usr/local/bin/gghstats

USER gghstats
WORKDIR /data
VOLUME /data
EXPOSE 8080

ENTRYPOINT ["gghstats", "serve"]
