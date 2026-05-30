package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

// ConvertRequest 转换请求参数
type ConvertRequest struct {
	Key        string `json:"key" binding:"required"`
	Resolution string `json:"resolution"`
	Framerate  int    `json:"framerate"`
	Quality    int    `json:"quality"`
}

// ConvertResponse 转换结果响应
type ConvertResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// ConvertProgress 转换进度
type ConvertProgress struct {
	Step    string `json:"step"`           // downloading | encoding | remuxing | uploading
	Percent int    `json:"percent"`        // 0-100
	Error   string `json:"error,omitempty"`
}

var (
	progressMap   = make(map[string]*ConvertProgress)
	progressMapMu sync.RWMutex
)

func setProgress(key, step string, percent int) {
	progressMapMu.Lock()
	defer progressMapMu.Unlock()
	progressMap[key] = &ConvertProgress{Step: step, Percent: percent}
}

func setError(key string, err string) {
	progressMapMu.Lock()
	defer progressMapMu.Unlock()
	progressMap[key] = &ConvertProgress{Error: err}
}

func getProgress(key string) *ConvertProgress {
	progressMapMu.RLock()
	defer progressMapMu.RUnlock()
	if p, ok := progressMap[key]; ok {
		return p
	}
	return nil
}

func clearProgress(key string) {
	progressMapMu.Lock()
	defer progressMapMu.Unlock()
	delete(progressMap, key)
}

// handleConvert 处理 .guac → MP4 转换请求（异步）
func handleConvert(c *gin.Context) {
	var req ConvertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ConvertResponse{Error: fmt.Sprintf("invalid request: %v", err)})
		return
	}

	// Check if already converting
	if p := getProgress(req.Key); p != nil && p.Error == "" {
		c.JSON(http.StatusOK, ConvertResponse{Success: true})
		return
	}

	setProgress(req.Key, "starting", 0)
	go doConvert(req.Key)
	c.JSON(http.StatusAccepted, ConvertResponse{Success: true})
}

func doConvert(key string) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "guac-convert-*")
	if err != nil {
		setError(key, fmt.Sprintf("failed to create temp dir: %v", err))
		return
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "input.guac")

	// Step 1: Download .guac from S3
	setProgress(key, "downloading", 5)
	guacKey := fmt.Sprintf("%s.guac", key)

	// Check if object exists first
	info, err := s3Client.StatObject(context.Background(), cfg.S3.Bucket, guacKey, minio.StatObjectOptions{})
	if err != nil {
		setError(key, fmt.Sprintf(".guac file not found in S3 (bucket=%s, key=%s, endpoint=%s): %v", cfg.S3.Bucket, guacKey, cfg.S3.Endpoint, err))
		return
	}
	log.Printf("Found .guac file: key=%s size=%d", guacKey, info.Size)

	obj, err := s3Client.GetObject(context.Background(), cfg.S3.Bucket, guacKey, minio.GetObjectOptions{})
	if err != nil {
		setError(key, fmt.Sprintf("failed to get .guac from s3: %v", err))
		return
	}
	defer obj.Close()

	guacData, err := io.ReadAll(obj)
	if err != nil {
		setError(key, fmt.Sprintf("failed to read .guac data: %v", err))
		return
	}

	if err := os.WriteFile(inputPath, guacData, 0644); err != nil {
		setError(key, fmt.Sprintf("failed to write input file: %v", err))
		return
	}
	setProgress(key, "downloading", 25)

	// Step 2: Run guacenc
	setProgress(key, "encoding", 25)
	guacArgs := []string{"-f", inputPath}
	cmd := exec.Command("guacenc", guacArgs...)
	cmd.Dir = tmpDir

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	timeout := getConvertTimeout()
	select {
	case err := <-done:
		if err != nil {
			setError(key, fmt.Sprintf("guacenc failed: %v", err))
			return
		}
	case <-time.After(timeout):
		cmd.Process.Kill()
		setError(key, fmt.Sprintf("guacenc timed out after %d seconds", int(timeout.Seconds())))
		return
	}
	setProgress(key, "encoding", 60)

	// Step 3: ffmpeg re-encode to H.264
	setProgress(key, "remuxing", 60)
	guacOutput := inputPath + ".m4v"
	h264Output := filepath.Join(tmpDir, "output.mp4")

	ffmpegCmd := exec.Command("ffmpeg", "-y", "-i", guacOutput,
		"-c:v", "libx264", "-pix_fmt", "yuv420p", "-movflags", "+faststart",
		h264Output)
	ffmpegDone := make(chan error, 1)
	go func() {
		ffmpegDone <- ffmpegCmd.Run()
	}()

	ffmpegTimeout := getConvertTimeout()
	select {
	case err := <-ffmpegDone:
		if err != nil {
			setError(key, fmt.Sprintf("ffmpeg re-encode failed: %v", err))
			return
		}
	case <-time.After(ffmpegTimeout):
		ffmpegCmd.Process.Kill()
		setError(key, fmt.Sprintf("ffmpeg timed out after %d seconds", int(ffmpegTimeout.Seconds())))
		return
	}
	setProgress(key, "remuxing", 85)

	// Step 4: Upload MP4 to S3
	setProgress(key, "uploading", 85)
	mp4Data, err := os.ReadFile(h264Output)
	if err != nil {
		setError(key, fmt.Sprintf("failed to read output file: %v", err))
		return
	}

	mp4Key := fmt.Sprintf("%s.mp4", key)
	_, err = s3Client.PutObject(context.Background(), cfg.S3.Bucket, mp4Key, bytes.NewReader(mp4Data), int64(len(mp4Data)), minio.PutObjectOptions{})
	if err != nil {
		setError(key, fmt.Sprintf("failed to upload mp4 to s3: %v", err))
		return
	}
	setProgress(key, "done", 100)

	// Clear progress after 5 minutes
	go func() {
		time.Sleep(5 * time.Minute)
		clearProgress(key)
	}()

	log.Printf("Conversion completed: key=%s", key)
}

// handleProgress 查询转换进度
func handleProgress(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
		return
	}
	p := getProgress(key)
	if p == nil {
		c.JSON(http.StatusOK, gin.H{"step": "", "percent": 0})
		return
	}
	c.JSON(http.StatusOK, p)
}
