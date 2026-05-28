# syntax=docker/dockerfile:1.7
#
# Multi-stage Go build → alpine runtime. Alpine over distroless because alpine
# bundles busybox (wget, nc) which lets compose healthcheck the HTTP port
# without us bolting a /healthz endpoint onto every server. ~8MB size penalty
# vs distroless/static, but operationally far easier to probe/debug.
#
# Base images pinned by digest for reproducibility — bump when intentional.

FROM golang:1.26-bookworm@sha256:386d475a660466863d9f8c766fec64d7fdad3edac2c6a05020c09534d71edb4b AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/musu-crawl .

FROM alpine:3.21@sha256:48b0309ca019d89d40f670aa1bc06e426dc0931948452e8491e3d65087abc07d
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /out/musu-crawl /usr/local/bin/musu-crawl
WORKDIR /app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/musu-crawl"]
