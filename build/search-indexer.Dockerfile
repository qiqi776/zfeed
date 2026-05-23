ARG GO_VERSION=1.26.2

FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src

RUN apk add --no-cache git ca-certificates
ENV GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
RUN go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
RUN go build -o /out/search-indexer ./app/rpc/search/search-indexer

FROM alpine:3.20
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=build /out/search-indexer /app/bin/search-indexer
COPY app/rpc/search/search-indexer/etc /app/app/rpc/search/search-indexer/etc
