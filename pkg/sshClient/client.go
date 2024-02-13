package sshClient

import (
	"fmt"
	"golang.org/x/crypto/ssh"
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

// Session session会话
func Session(client *ssh.Client, cols, rows int) (*ssh.Session, error) {
	session, err := client.NewSession()
	if err != nil {
		logger.Error(fmt.Sprintf("创建session失败-%s", err.Error()))
		return nil, err
	}
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // disable echo
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	if err = session.RequestPty("xterm", rows, cols, modes); err != nil {
		logger.Error(fmt.Sprintf("重新定义窗口大小失败-%s", err.Error()))
		return nil, err
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
