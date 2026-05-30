# ---- Build stage ----
FROM docker.1ms.run/library/golang:1.25.10 AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOPROXY=https://goproxy.cn,direct

WORKDIR /opt
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o guac-worker main.go

# ---- Runtime stage ----
FROM docker.1ms.run/litios/guacenc:latest
COPY --from=builder /opt/guac-worker /usr/local/bin/guac-worker
COPY config.yaml /app/config.yaml
RUN mkdir -p /recordings
WORKDIR /app
VOLUME /recordings
EXPOSE 8080
ENTRYPOINT ["guac-worker"]
