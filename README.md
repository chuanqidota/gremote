# webssh-go
go版本实现的webssh后端

### 设计思路见
draw.io>github>webssh-go

### 启动前准备
#### 配置config.
##### 1、启动redis
##### 2、启动es
##### 3、启动minio >  需要设置桶名、密钥、权限(可读)


### 测试思路-终端：
##### 1、ws/v1/obtain-key 获取key
##### 2、ws/v1/{key} 连接终端
##### 3、发送命令
##### 4、ws/v1/record-url 获取回放的url

