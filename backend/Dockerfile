FROM golang:1.21.6 AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOPROXY=https://goproxy.cn,direct

WORKDIR /opt

# 先复制 go.mod/go.sum，利用 Docker 层缓存
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o gwebssh main.go

FROM alpine:3

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /opt
COPY --from=builder /opt/gwebssh /opt/gwebssh
COPY --from=builder /opt/config /opt/config

EXPOSE 8000

ENTRYPOINT ["./gwebssh"]
