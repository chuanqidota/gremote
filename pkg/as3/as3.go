package as3

import (
	"bytes"
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"webssh-go/config"
	"webssh-go/pkg/logger"
)

var As3Client *minio.Client

func Init() {
	endpoint := config.Conf.As3.EndPoint
	accessKeyID := config.Conf.As3.AccessKeyID
	secretAccessKey := config.Conf.As3.SecretAccessKey
	useSSL := config.Conf.As3.UseSSL
	// 初始化MinIO Go客户端对象
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		logger.Error(fmt.Sprintf("初始化as3客户端失败-%s", err.Error()))
		return
	}
	As3Client = client
	logger.Info("初始化as3客户端成功")
}

// UploadFile 上传数据到as3中，文件名key
func UploadFile(key string, data []byte) {
	//jsonData, err := json.Marshal(data)
	//if err != nil {
	//	logger.Error(fmt.Sprintf("上传As3前解析文件失败-%s", err.Error()))
	//}
	BucketName := config.Conf.As3.Bucket
	_, err := As3Client.PutObject(context.Background(), BucketName, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{})
	//_, err = As3Client.PutObject(context.Background(), BucketName, key, bytes.NewReader(jsonData), int64(len(jsonData)), minio.PutObjectOptions{})
	if err != nil {
		logger.Error(fmt.Sprintf("上传As3文件失败-%s", err.Error()))
	}
}
