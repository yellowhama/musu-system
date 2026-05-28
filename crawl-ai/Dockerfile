# syntax=docker/dockerfile:1.7
#
# Multi-stage Go build → alpine runtime. Alpine over distroless because alpine
# bundles busybox (wget, nc) which lets compose healthcheck the HTTP port
# without us bolting a /healthz endpoint onto every server. ~8MB size penalty
# vs distroless/static, but operationally far easier to probe/debug.
#
# Build context: monorepo root (`docker build -f crawl-ai/Dockerfile .` from
# the musu-system root). GOWORK=off so Docker builds rely on each go.mod's
# `replace github.com/yellowhama/musu-system/core => ../core` directive and
# don't need the workspace file. Outside Docker, the root go.work supplies
# the same wiring transparently.
#
# Base images pinned by digest for reproducibility — bump when intentional.

FROM golang:1.26-bookworm@sha256:386d475a660466863d9f8c766fec64d7fdad3edac2c6a05020c09534d71edb4b AS build
WORKDIR /src

# Step 1: copy only manifests for dependency-download layer caching.
COPY core/go.mod core/go.sum ./core/
COPY crawl-ai/go.mod crawl-ai/go.sum ./crawl-ai/

WORKDIR /src/crawl-ai
ENV GOWORK=off
RUN go mod download

# Step 2: copy actual source and build.
WORKDIR /src
COPY core ./core
COPY crawl-ai ./crawl-ai

WORKDIR /src/crawl-ai
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/musu-crawl .

FROM alpine:3.21@sha256:48b0309ca019d89d40f670aa1bc06e426dc0931948452e8491e3d65087abc07d
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /out/musu-crawl /usr/local/bin/musu-crawl
WORKDIR /app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/musu-crawl"]
