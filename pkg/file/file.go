package file

import (
	"fmt"
	"webssh-go/app/api/params"

	"webssh-go/pkg/logger"

	"io"
	"mime/multipart"
	"path/filepath"

	"bytes"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type fileHandle struct {
}

var FileHandle = new(fileHandle)

// ListFile 查看文件列表
func (f *fileHandle) ListFile(info params.Info, path string) ([]map[string]any, error) {
	// 使用切片嵌套的map来存储目录和文件的大小和名称
	result := make([]map[string]interface{}, 0)

	// 初始化登录信息
	target := info.Target
	username := info.Username
	password := info.Password
	port := info.Port

	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
			// 或者使用SSH密钥：ssh.PublicKeys(privateKey),
		},
		// 可以添加其他配置项，如HostKeyCallback等
	}

	// 建立SSH连接
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", target, port), sshConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("建立ssh连接失败-%s", err.Error()))
		return result, err
	}
	defer client.Close()

	// 使用SFTP连接
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		logger.Error(fmt.Sprintf("sftp客户端连接失败-%s", err.Error()))
		return result, err
	}
	defer sftpClient.Close()

	// 读取远程目录中的文件信息
	fileInfos, err := sftpClient.ReadDir(path)
	if err != nil {
		logger.Error(fmt.Sprintf("获取远程路径下的文件失败-%s", err.Error()))
		return result, err
	}

	// 遍历文件信息并输出目录和文件信息
	for _, fileInfo := range fileInfos {
		entry := make(map[string]interface{})
		entry["name"] = fileInfo.Name()
		entry["size"] = fileInfo.Size()

		if fileInfo.IsDir() {
			// 如果是目录，标记为目录类型
			entry["type"] = "directory"
		} else {
			// 如果是文件，标记为文件类型
			entry["type"] = "file"
		}

		// 将当前文件信息添加到切片中
		result = append(result, entry)
	}
	return result, nil

}

// UploadFile 上传文件
func (f *fileHandle) UploadFile(file *multipart.FileHeader, info params.Info, path string) error {
	// 初始化登录信息
	target := info.Target
	username := info.Username
	password := info.Password
	port := info.Port

	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
			// 或者使用SSH密钥：ssh.PublicKeys(privateKey),
		},
		// 可以添加其他配置项，如HostKeyCallback等
	}

	// 建立SSH连接
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", target, port), sshConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("建立ssh连接失败-%s", err.Error()))
		return err
	}
	defer client.Close()

	// 使用SFTP连接
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		logger.Error(fmt.Sprintf("sftp客户端连接失败-%s", err.Error()))
		return err
	}
	defer sftpClient.Close()

	// 打开远程文件以写入
	remoteFilePath := filepath.Join(path, file.Filename)
	remoteFile, err := sftpClient.Create(remoteFilePath)
	if err != nil {
		logger.Error(fmt.Sprintf("创建文件失败-%s", err.Error()))
		return err
	}
	defer remoteFile.Close()

	// 打开本地上传的文件以读取
	localFile, err := file.Open()
	if err != nil {
		logger.Error(fmt.Sprintf("读取文件失败-%s", err.Error()))
		return err
	}
	defer localFile.Close()

	// 将本地文件内容复制到远程文件
	_, err = io.Copy(remoteFile, localFile)
	if err != nil {
		logger.Error("复制文件失败")
		return err
	}
	return nil
}

// DownLoadFile 下载文件
func (f *fileHandle) DownLoadFile(info params.Info, path string, filename string) ([]byte, error) {
	target := info.Target
	username := info.Username
	password := info.Password
	port := info.Port

	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
			// 或者使用SSH密钥：ssh.PublicKeys(privateKey),
		},
		// 可以添加其他配置项，如HostKeyCallback等
	}

	// 建立SSH连接
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", target, port), sshConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("建立ssh连接失败-%s", err.Error()))
		return nil, err
	}
	defer client.Close()

	// 使用SFTP连接
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		logger.Error(fmt.Sprintf("sftp客户端连接失败-%s", err.Error()))
		return nil, err
	}
	defer sftpClient.Close()

	// 指定远程文件路径
	remoteFilePath := filepath.Join(path, filename)
	// 打开远程文件以读取
	remoteFile, err := sftpClient.Open(remoteFilePath)
	if err != nil {
		logger.Error(fmt.Sprintf("读取远程文件失败-%s", err.Error()))
		return nil, err
	}
	defer remoteFile.Close()

	// 将远程文件内容直接写入HTTP响应
	buffer := new(bytes.Buffer)
	_, err = io.Copy(buffer, remoteFile)
	if err != nil {
		logger.Error(fmt.Sprintf("写入http文件失败-%s", err.Error()))
		return nil, err
	}
	return buffer.Bytes(), nil

}
