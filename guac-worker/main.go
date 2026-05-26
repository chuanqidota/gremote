package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

type Config struct {
	S3Endpoint  string
	S3AccessKey string
	S3SecretKey string
	S3Bucket    string
	S3UseSSL    bool
}

var cfg Config

func main() {
	cfg = Config{
		S3Endpoint:  getEnv("S3_ENDPOINT", "minio:9000"),
		S3AccessKey: getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey: getEnv("S3_SECRET_KEY", ""),
		S3Bucket:    getEnv("S3_BUCKET", "gwebssh"),
		S3UseSSL:    os.Getenv("S3_USE_SSL") == "true",
	}

	r := gin.Default()

	r.POST("/convert", handleConvert)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	log.Println("guac-worker starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
