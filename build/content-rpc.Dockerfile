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
RUN go build -o /out/content-rpc ./app/rpc/content

FROM alpine:3.20
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=build /out/content-rpc /app/bin/content-rpc
COPY app/rpc/content/etc /app/app/rpc/content/etc
RUN mkdir -p /var/log/ran-feed

EXPOSE 5001 5005 9291
