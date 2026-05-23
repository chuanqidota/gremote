package s3

import (
	"bytes"
	"context"
	"fmt"
	"gwebssh/config"
	"gwebssh/pkg/logger"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var S3Client *minio.Client

func Init() {
	endpoint := config.Conf.S3.EndPoint
	accessKeyID := config.Conf.S3.AccessKeyID
	secretAccessKey := config.Conf.S3.SecretAccessKey
	useSSL := config.Conf.S3.UseSSL
	// 初始化MinIO Go客户端对象
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		logger.Error(fmt.Sprintf("初始化S3客户端失败-%s", err.Error()))
		return
	}
	S3Client = client
	logger.Info("初始化S3客户端成功")
}

// UploadFile 上传数据到S3中，文件名key
func UploadFile(key string, data []byte) error {
	if S3Client == nil {
		err := fmt.Errorf("S3客户端未初始化")
		logger.Error(err.Error())
		return err
	}
	BucketName := config.Conf.S3.Bucket
	_, err := S3Client.PutObject(context.Background(), BucketName, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{})
	if err != nil {
		logger.Error(fmt.Sprintf("上传S3文件失败-%s", err.Error()))
		return err
	}
	return nil
}

// GetFile 从S3中读取文件内容
func GetFile(key string) ([]byte, error) {
	if S3Client == nil {
		return nil, fmt.Errorf("S3客户端未初始化")
	}
	BucketName := config.Conf.S3.Bucket
	obj, err := S3Client.GetObject(context.Background(), BucketName, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("读取S3文件失败-%s", err.Error())
	}
	defer obj.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(obj)
	if err != nil {
		return nil, fmt.Errorf("读取S3文件流失败-%s", err.Error())
	}
	return buf.Bytes(), nil
}
