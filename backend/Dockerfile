FROM golang:1.21.6 AS builder
RUN go env -w GO111MODULE=on \
    && go env -w CGO_ENABLED=0 \
    && go env -w GOOS=linux \
    && go env -w GOPROXY=https://goproxy.cn,direct
RUN mkdir -p /opt
WORKDIR /opt
COPY . .
RUN go mod tidy
RUN go build -o gwebssh main.go

FROM alpine:3
RUN mkdir -p /opt
WORKDIR /opt
COPY --from=builder /opt/gwebssh /opt/gwebssh
COPY --from=builder /opt/config /opt/config
RUN chmod +x /opt/gwebssh
EXPOSE 8000

ENTRYPOINT ["/opt/gwebssh"]