# syntax=docker/dockerfile:1.7
# Multi-stage Go build → alpine runtime. See crawl-ai/Dockerfile for the
# rationale of alpine vs distroless and the monorepo build-context handling.
# Base images pinned by digest.

FROM golang:1.26-bookworm@sha256:386d475a660466863d9f8c766fec64d7fdad3edac2c6a05020c09534d71edb4b AS build
WORKDIR /src

COPY core/go.mod core/go.sum ./core/
COPY nurikun/go.mod nurikun/go.sum ./nurikun/

WORKDIR /src/nurikun
ENV GOWORK=off
RUN go mod download

WORKDIR /src
COPY core ./core
COPY nurikun ./nurikun

WORKDIR /src/nurikun
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/musu-nurikun .

FROM alpine:3.21@sha256:48b0309ca019d89d40f670aa1bc06e426dc0931948452e8491e3d65087abc07d
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /out/musu-nurikun /usr/local/bin/musu-nurikun
WORKDIR /app
EXPOSE 8088
ENTRYPOINT ["/usr/local/bin/musu-nurikun"]
