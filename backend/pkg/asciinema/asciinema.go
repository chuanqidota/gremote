package asciinema

import (
	"encoding/json"
	"fmt"
	"time"
	"gwebssh/app/ws/utils/recordAudit"
)

// asciinema 文档https://docs.asciinema.org/manual/asciicast/v2/

type RecHeader struct {
	Version   int   `json:"version"`
	Width     int   `json:"width"`
	Height    int   `json:"height"`
	Timestamp int64 `json:"timestamp"`
	Env       struct {
		Shell string `json:"SHELL"`
		Term  string `json:"TERM"`
	} `json:"env"`
}

// WriteHeader 写头部信息
func WriteHeader(key string, cols, rows int, startTime time.Time, record *recordAudit.EsRecord) {
	header := RecHeader{
		Version:   2,
		Width:     cols,
		Height:    rows,
		Timestamp: startTime.Unix(),
	}
	header.Env.Shell = "/bin/bash"
	header.Env.Term = "xterm-256color"
	history, _ := json.Marshal(header)
	data := map[string]any{
		"key":       key,
		"timeStamp": time.Now().UnixNano() / int64(time.Millisecond),
		"history":   string(history),
	}
	record.WriteData(data)
}

// WriteData 写入数据
func WriteData(key string, startTime time.Time, out string, record *recordAudit.EsRecord) {
	sub := float64(time.Since(startTime).Microseconds()) / float64(1000000)
	history, _ := json.Marshal([]any{sub, "o", out})
	data := map[string]any{
		"key":       key,
		"timeStamp": time.Now().UnixNano() / int64(time.Millisecond),
		"history":   string(history),
	}
	record.WriteData(data)
}

// WriteSize 写尺寸数据
func WriteSize(key string, startTime time.Time, cols, rows int, record *recordAudit.EsRecord) {
	sub := float64(time.Since(startTime).Microseconds()) / float64(1000000)
	history, _ := json.Marshal([]any{sub, "r", fmt.Sprintf("%d*%d", cols, rows)})
	data := map[string]any{
		"key":       key,
		"timeStamp": time.Now().UnixNano() / int64(time.Millisecond),
		"history":   string(history),
	}
	record.WriteData(data)
}
