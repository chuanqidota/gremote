FROM golang:1.21.6 AS builder
RUN go env -w GO111MODULE=on \
    && go env -w CGO_ENABLED=0 \
    && go env -w GOOS=linux \
    && go env -w GOPROXY=https://goproxy.cn,direct
RUN mkdir -p /opt
WORKDIR /opt
COPY . .
RUN go mod tidy
RUN go build -o webssh main.go

FROM alpine:3
RUN mkdir -p /opt
WORKDIR /root/data
COPY --from=builder /opt/webssh /opt/webssh
RUN chmod +x webssh
EXPOSE 8080

ENTRYPOINT ["./webssh"]