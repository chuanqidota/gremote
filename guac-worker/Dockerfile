FROM docker.1ms.run/library/golang:1.25.10 AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o guac-worker .

FROM docker.1ms.run/litios/guacenc:latest
COPY --from=go-builder /app/guac-worker /usr/local/bin/guac-worker
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /recordings
VOLUME /recordings

EXPOSE 8080
ENTRYPOINT ["guac-worker"]
