package terminal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
	"io"
	"sync"
	"time"
	"gwebssh/app/ws/utils/recordAudit"
	"gwebssh/pkg/asciinema"
	"gwebssh/pkg/logger"
)

// Client ssh客户端
func Client(username, password, target string, port int) (*ssh.Client, error) {
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
		Config: ssh.Config{
			// 默认加密方式 aes128-ctr aes192-ctr aes256-ctr aes128-gcm@openssh.com arcfour256 arcfour128
			// 连 linux 通常没有问题，但是很多交换机其实默认只提供 aes128-cbc 3des-cbc aes192-cbc aes256-cbc 这些。
			Ciphers: []string{"aes128-ctr", "aes192-ctr", "aes256-ctr", "aes128-gcm@openssh.com", "arcfour256", "arcfour128", "aes128-cbc", "3des-cbc", "aes192-cbc", "aes256-cbc"},
		},
	}
	// 建立SSH连接
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", target, port), sshConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("建立ssh连接失败-%s", err.Error()))
		return nil, err
	}
	return client, nil
}

// wsBufferWriter 定义一个输出和出错的结构体
type wsBufferWriter struct {
	buffer bytes.Buffer
	mu     sync.Mutex
}

// implement Write interface to write bytes from ssh server into bytes.Buffer.
func (w *wsBufferWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.Write(p)
}

// Terminal 终端
type Terminal struct {
	Session     *ssh.Session
	StdinPipe   io.WriteCloser
	ComboOutput *wsBufferWriter
}

// NewTerminal 初始化终端
func NewTerminal(client *ssh.Client, cols, rows int) (*Terminal, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	// 标准输出和错误输出 存入到wsBufferWriter
	comboWriter := new(wsBufferWriter)
	session.Stdout = comboWriter
	session.Stderr = comboWriter

	stdinPipe, err := session.StdinPipe()
	if err != nil {
		return nil, err
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     //  禁用回显（0禁用，1启动） 一定要开，不然会出问题
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud  传输速率
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	if err = session.RequestPty("xterm", cols, rows, modes); err != nil {
		return nil, err
	}
	if err := session.Shell(); err != nil {
		logger.Error(fmt.Sprintf("启动shell失败呢-%s", err.Error()))
		return nil, err
	}
	terminal := Terminal{
		Session:     session,
		StdinPipe:   stdinPipe,
		ComboOutput: comboWriter,
	}
	return &terminal, nil
}

// Close 关闭终端
func (t *Terminal) Close() {
	if t.Session != nil {
		err := t.Session.Close()
		logger.Error("session关闭失败-%s", err.Error())
	}
}

// ReceiveWsMsg 接受ws消息 发送到terminal
func (t *Terminal) ReceiveWsMsg(ws *websocket.Conn, quitChan chan bool, key string, startTime time.Time, record *recordAudit.EsRecord) {
	defer setQuit(quitChan)
	for {
		select {
		case <-quitChan:
			return
		default:
			// 接受ws消息
			_, message, err := ws.ReadMessage()
			if err != nil {
				_ = ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("读取信息出错-%s", err.Error())))
				return
			}
			// 解析ws消息
			var data map[string]any
			err = json.Unmarshal(message, &data)
			if err != nil {
				_, err = t.StdinPipe.Write(message)
			} else {
				resize, resizeOk := data["resize"]
				if resizeOk {
					resize_, _ := resize.([]any)
					cols := int(resize_[0].(float64))
					rows := int(resize_[1].(float64))
					err = t.Session.WindowChange(cols, rows)
					if err != nil {
						_ = ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("调整窗口大小出错-%s", err.Error())))
						return
					}
				}
			}
		}
	}
}

// WriteWsMsg 读取终端输出往ws中写消息
func (t *Terminal) WriteWsMsg(ws *websocket.Conn, quitChan chan bool, esDataChan chan []byte) {
	defer setQuit(quitChan)
	for {
		select {
		case <-quitChan:
			return
		default:
			if t.ComboOutput.buffer.Len() != 0 {
				// 往ws中输出
				_ = ws.WriteMessage(websocket.TextMessage, t.ComboOutput.buffer.Bytes())
				// 把操作记录写到es中
				esDataChan <- t.ComboOutput.buffer.Bytes()
				// 重置ComboOutput的缓冲区
				t.ComboOutput.buffer.Reset()
			}
		}
	}
}

// WriteEsData 写入到数据到es中
func (t *Terminal) WriteEsData(quitChan chan bool, key string, startTime time.Time, record *recordAudit.EsRecord, esDataChan chan []byte) {
	defer setQuit(quitChan)
	for {
		select {
		case <-quitChan:
			return
		case data := <-esDataChan:
			// 将数据写入 ES
			asciinema.WriteData(key, startTime, string(data), record)
		}
	}
}

// SessionWait 等待session结束
func (t *Terminal) SessionWait(quitChan chan bool) {
	defer setQuit(quitChan)
	select {
	case <-quitChan:
		return
	default:
		if err := t.Session.Wait(); err != nil {
			logger.Error("session-wait失败-%s", err.Error())
		}
	}
}

func setQuit(ch chan bool) {
	ch <- true
}
