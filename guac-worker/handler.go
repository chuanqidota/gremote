package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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

// handleConvert 处理 .guac → MP4 转换请求
// 流程：S3 下载 .guac → guacenc 转 .m4v → ffmpeg 转 H.264 MP4 → 上传 S3
func handleConvert(c *gin.Context) {
	var req ConvertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ConvertResponse{Error: fmt.Sprintf("invalid request: %v", err)})
		return
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "guac-convert-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, ConvertResponse{Error: fmt.Sprintf("failed to create temp dir: %v", err)})
		return
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "input.guac")

	// Download .guac from S3
	guacKey := fmt.Sprintf("%s.guac", req.Key)
	obj, err := s3Client.GetObject(context.Background(), cfg.S3.Bucket, guacKey, minio.GetObjectOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ConvertResponse{Error: fmt.Sprintf("failed to get .guac from s3: %v", err)})
		return
	}
	defer obj.Close()

	guacData, err := io.ReadAll(obj)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ConvertResponse{Error: fmt.Sprintf("failed to read .guac data: %v", err)})
		return
	}

	// Write .guac file
	if err := os.WriteFile(inputPath, guacData, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, ConvertResponse{Error: fmt.Sprintf("failed to write input file: %v", err)})
		return
	}

	// Build guacenc command (output is auto-named {input}.m4v)
	guacArgs := []string{"-f"}
	if req.Resolution != "" {
		guacArgs = append(guacArgs, "-s", req.Resolution)
	}
	if req.Framerate > 0 {
		guacArgs = append(guacArgs, "-r", fmt.Sprintf("%d", req.Framerate))
	}
	guacArgs = append(guacArgs, inputPath)

	// Run guacenc with timeout
	cmd := exec.Command("guacenc", guacArgs...)
	cmd.Dir = tmpDir

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			c.JSON(http.StatusInternalServerError, ConvertResponse{Error: fmt.Sprintf("guacenc failed: %v", err)})
			return
		}
	case <-time.After(5 * time.Minute):
		cmd.Process.Kill()
		c.JSON(http.StatusGatewayTimeout, ConvertResponse{Error: "guacenc conversion timed out after 5 minutes"})
		return
	}

	// guacenc outputs MPEG-4 Part 2 (.m4v), re-encode to H.264 for browser compatibility
	guacOutput := inputPath + ".m4v"
	h264Output := filepath.Join(tmpDir, "output.mp4")

	ffmpegCmd := exec.Command("ffmpeg", "-y", "-i", guacOutput,
		"-c:v", "libx264", "-pix_fmt", "yuv420p", "-movflags", "+faststart",
		h264Output)
	ffmpegDone := make(chan error, 1)
	go func() {
		ffmpegDone <- ffmpegCmd.Run()
	}()

	select {
	case err := <-ffmpegDone:
		if err != nil {
			c.JSON(http.StatusInternalServerError, ConvertResponse{Error: fmt.Sprintf("ffmpeg re-encode failed: %v", err)})
			return
		}
	case <-time.After(5 * time.Minute):
		ffmpegCmd.Process.Kill()
		c.JSON(http.StatusGatewayTimeout, ConvertResponse{Error: "ffmpeg re-encode timed out after 5 minutes"})
		return
	}

	// Read H.264 MP4
	mp4Data, err := os.ReadFile(h264Output)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ConvertResponse{Error: fmt.Sprintf("failed to read output file: %v", err)})
		return
	}

	// Upload MP4 to S3
	mp4Key := fmt.Sprintf("%s.mp4", req.Key)
	_, err = s3Client.PutObject(context.Background(), cfg.S3.Bucket, mp4Key, bytes.NewReader(mp4Data), int64(len(mp4Data)), minio.PutObjectOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ConvertResponse{Error: fmt.Sprintf("failed to upload mp4 to s3: %v", err)})
		return
	}

	c.JSON(http.StatusOK, ConvertResponse{Success: true})
}
