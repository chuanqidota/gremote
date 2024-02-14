package sshClient

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"log"
	"sync"
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
	}
	// 建立SSH连接
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", target, port), sshConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("建立ssh连接失败-%s", err.Error()))
		return nil, err
	}
	return client, nil
}

// wsBufferWriter 缓存
type wsBufferWriter struct {
	buffer bytes.Buffer
	mu     sync.Mutex
}

// Write 实现Writer接口
func (w *wsBufferWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.Write(p)
}

// Session session会话
func Session(client *ssh.Client, cols, rows int) (*ssh.Session, error) {
	session, err := client.NewSession()
	if err != nil {
		logger.Error(fmt.Sprintf("创建session失败-%s", err.Error()))
		return nil, err
	}
	// 用wsBufferWriter接受缓存
	comboWriter := new(wsBufferWriter)
	session.Stdout = comboWriter
	session.Stderr = comboWriter
	// 终端模式
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     //  禁用回显（0禁用，1启动）
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud  传输速率
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	if err = session.RequestPty("xterm", rows, cols, modes); err != nil {
		logger.Error(fmt.Sprintf("重新定义窗口大小失败-%s", err.Error()))
		return nil, err
	}
	if err = session.Shell(); err != nil {
		log.Fatalf("start shell error: %s", err.Error())
	}
	return session, nil
}

// Write 方法用于从远程服务器读取响应
func Write(session *ssh.Session, cmd string) ([]byte, error) {
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		logger.Error(fmt.Sprintf("读取远程服务器响应失败-%s", err.Error()))
		return nil, err
	}
	return output, nil
}
