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
