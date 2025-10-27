# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . ./
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -o /out/server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -H -u 10001 app
USER app
WORKDIR /app
COPY --from=build /out/server /app/server
VOLUME ["/records"]
ENV HTTP_ADDR=:8080 \
    RECORD_DIR=/records
EXPOSE 8080
ENTRYPOINT ["/app/server"]
