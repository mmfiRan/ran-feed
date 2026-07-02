ARG GO_VERSION=1.25

FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src

RUN apk add --no-cache git ca-certificates
ENV GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
RUN go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
RUN go build -o /out/search-rpc ./app/rpc/search

FROM alpine:3.20
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=build /out/search-rpc /app/bin/search-rpc
COPY app/rpc/search/etc /app/app/rpc/search/etc
RUN mkdir -p /var/log/ran-feed

EXPOSE 5006 9296