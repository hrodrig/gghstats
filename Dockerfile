# Multi-stage build for local use (make docker-build). Release images use Dockerfile.release + GoReleaser.
FROM golang:1.26.2-alpine AS builder

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

# Alpine 3.21 keeps OpenSSL 3.3.x (see 3.22+ for 3.5.x). apk upgrade pulls patches from the 3.21 repo.
FROM alpine:3.21
RUN apk update \
    && apk add --no-cache ca-certificates \
    && apk upgrade --no-cache \
    && addgroup -g 1000 gghstats \
    && adduser -D -u 1000 -G gghstats -h /data gghstats
COPY --from=builder /app/gghstats /usr/local/bin/gghstats

USER gghstats
WORKDIR /data
VOLUME /data
EXPOSE 8080

ENTRYPOINT ["gghstats", "serve"]
