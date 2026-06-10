# Multi-stage build for local use (make docker-build). Release images use Dockerfile.release + GoReleaser.
FROM golang:1.26.4-alpine AS builder

# Pin module proxy so builds do not inherit a broken or empty GOPROXY from the host/BuildKit environment.
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.org

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

# Alpine 3.22+: fresher busybox/ca-certificates vs Grype noise on 3.21; apk upgrade pulls security revisions.
FROM alpine:3.22
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
