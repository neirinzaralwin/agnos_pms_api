# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/api \
    ./cmd/api

FROM alpine:3.21

RUN apk add --no-cache ca-certificates wget \
    && adduser -D -u 65532 -g nonroot nonroot

WORKDIR /

COPY --from=builder /out/api /api

USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/api"]
