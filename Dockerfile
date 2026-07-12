# syntax=docker/dockerfile:1
# Local / CI image (make docker-build, docker compose). Release images: Dockerfile.release + GoReleaser.
# Runtime: distroless static Debian 13 (no shell/apk — smaller attack surface), same pattern as groot.
FROM --platform=$BUILDPLATFORM golang:1.26.5-bookworm AS builder

# Pin module proxy so builds do not inherit a broken or empty GOPROXY from the host/BuildKit environment.
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.org

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILDDATE=unknown

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags "-s -w \
    -X 'github.com/hrodrig/gghstats/internal/version.Version=${VERSION}' \
    -X 'github.com/hrodrig/gghstats/internal/version.Commit=${COMMIT}' \
    -X 'github.com/hrodrig/gghstats/internal/version.BuildDate=${BUILDDATE}'" \
    -o /out/gghstats ./cmd/gghstats

FROM gcr.io/distroless/static-debian13:nonroot
LABEL org.opencontainers.image.title="gghstats"
LABEL org.opencontainers.image.description="GitHub traffic dashboard and CLI"
LABEL org.opencontainers.image.source="https://github.com/hrodrig/gghstats"
WORKDIR /data
# Ensure /data is writable by nonroot (UID 65532) for SQLite.
COPY --chown=65532:65532 <<EOF /data/.keep
EOF
COPY --from=builder /out/gghstats /usr/local/bin/gghstats
ENV GGHSTATS_DB=/data/gghstats.db
USER nonroot:nonroot
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/gghstats", "serve"]
