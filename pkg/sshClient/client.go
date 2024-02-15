package sshClient

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"time"
	"webssh-go/pkg/logger"
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

// Session session会话
func Session(client *ssh.Client) (*ssh.Session, error) {
	session, err := client.NewSession()
	if err != nil {
		logger.Error(fmt.Sprintf("创建session失败-%s", err.Error()))
		return nil, err
	}
	return session, nil
}

// Resize 重新定义窗口大小
func Resize(session *ssh.Session, cols, rows int) error {
	if err := session.WindowChange(cols, rows); err != nil {
		return err
	}
	return nil
}

// Terminal 启动一个终端
func Terminal(session *ssh.Session, stdout, stderr io.Writer, stdin io.Reader, cols, rows int) error {
	session.Stdout = io.MultiWriter(os.Stdout, stdout)
	session.Stderr = io.MultiWriter(os.Stderr, stderr)
	session.Stdin = stdin
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     //  禁用回显（0禁用，1启动）
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud  传输速率
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	if err := session.RequestPty("xterm", cols, rows, modes); err != nil {
		return err
	}
	if err := session.Shell(); err != nil {
		logger.Error(fmt.Sprintf("启动shell失败呢-%s", err.Error()))
	}
	if err := session.Wait(); err != nil {
		logger.Error(fmt.Sprintf("session响应失败-%s", err.Error()))
	}
	return nil
}

// StdOutErr 定义标准输出和标准错误输出
type StdOutErr struct {
	Conn *websocket.Conn
}

// Write 实现io.Writer这个接口-往websocket中写入消息
func (s *StdOutErr) Write(p []byte) (n int, err error) {
	err = s.Conn.WriteMessage(websocket.TextMessage, p)
	return len(p), err
}

// StdIn 定义标准输入
type StdIn struct {
	Conn    *websocket.Conn
	Session *ssh.Session
}

// Read 实现io.Reader接口，从websocket中读取消息
func (s *StdIn) Read(p []byte) (n int, err error) {
	_, message, err := s.Conn.ReadMessage()
	if err != nil {
		return 0, err
	}

	var data map[string]any
	err = json.Unmarshal(message, &data)
	if err != nil {
		return 0, err
	}
	resize, resizeOk := data["resize"]
	if resizeOk {
		resize, _ := resize.([]int)
		cols := resize[0]
		rows := resize[1]
		if err = Resize(s.Session, cols, rows); err != nil {
			return 0, err
		}
	}
	text, textOk := data["data"]
	if textOk {
		text, _ := text.(string)
		n = copy(p, []byte(fmt.Sprintf("%s\n", text)))
		return n, err
	}
	return 0, err
}
