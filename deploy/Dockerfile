# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS builder
WORKDIR /src

COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o /out/proxy-server ./cmd/server

FROM alpine:3.20
RUN addgroup -S app && adduser -S -G app app
WORKDIR /app
RUN mkdir -p /app/data && chown -R app:app /app
COPY --from=builder /out/proxy-server /app/proxy-server
USER app
EXPOSE 8899 1080
ENTRYPOINT ["/app/proxy-server"]
