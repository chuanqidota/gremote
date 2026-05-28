package minio

import (
	"bytes"
	"context"
	"fmt"
	"gremote/config"
	"gremote/pkg/logger"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var S3Client *minio.Client

func Init() {
	endpoint := config.Conf.S3.Endpoint
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

	// Auto-create bucket if not exists
	bucketName := config.Conf.S3.Bucket
	exists, err := client.BucketExists(context.Background(), bucketName)
	if err != nil {
		logger.Error(fmt.Sprintf("检查S3 bucket失败-%s", err.Error()))
	} else if !exists {
		if err := client.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{}); err != nil {
			logger.Error(fmt.Sprintf("创建S3 bucket失败-%s", err.Error()))
		} else {
			logger.Info(fmt.Sprintf("创建S3 bucket成功: %s", bucketName))
		}
	}
}

// UploadFile 上传数据到S3中，文件名key
func UploadFile(key string, data []byte) error {
	if S3Client == nil {
		err := fmt.Errorf("S3客户端未初始化")
		logger.Error(err.Error())
		return err
	}
	bucketName := config.Conf.S3.Bucket
	_, err := S3Client.PutObject(context.Background(), bucketName, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{})
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
	bucketName := config.Conf.S3.Bucket
	obj, err := S3Client.GetObject(context.Background(), bucketName, key, minio.GetObjectOptions{})
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

// ListFiles 列出S3中指定前缀的文件
func ListFiles(prefix string) ([]string, error) {
	if S3Client == nil {
		return nil, fmt.Errorf("S3客户端未初始化")
	}
	bucketName := config.Conf.S3.Bucket
	var files []string
	for obj := range S3Client.ListObjects(context.Background(), bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}) {
		if obj.Err != nil {
			return nil, fmt.Errorf("列出S3文件失败-%s", obj.Err)
		}
		files = append(files, obj.Key)
	}
	return files, nil
}

// GetFileSize 获取S3文件大小（字节），不下载文件内容
func GetFileSize(key string) (int64, error) {
	if S3Client == nil {
		return 0, fmt.Errorf("S3客户端未初始化")
	}
	bucketName := config.Conf.S3.Bucket
	info, err := S3Client.StatObject(context.Background(), bucketName, key, minio.StatObjectOptions{})
	if err != nil {
		return 0, fmt.Errorf("获取S3文件信息失败-%s", err.Error())
	}
	return info.Size, nil
}
