package asciinema

import (
	"encoding/json"
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

// WriteInputData 写入用户输入数据
func WriteInputData(key string, startTime time.Time, input string, record *recordAudit.EsRecord) {
	sub := float64(time.Since(startTime).Microseconds()) / float64(1000000)
	history, _ := json.Marshal([]any{sub, "i", input})
	data := map[string]any{
		"key":       key,
		"timeStamp": time.Now().UnixNano() / int64(time.Millisecond),
		"history":   string(history),
	}
	record.WriteData(data)
}

