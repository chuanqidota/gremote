# 后端配置与项目结构重构设计

## Context

后端审计发现配置文件存在字段命名不一致、typo、日志泄露密码等问题；项目结构存在死代码、空文件、包模式不统一等问题。本次重构目标是修复配置层和项目结构层的所有已知问题，为后续代码质量和安全改进打好基础。

---

## 1. 配置字段命名与 Typo 修复

### 1.1 Config 结构体改为命名类型

**文件:** `config/config.go`

将匿名嵌套结构体改为具名类型，使其可独立引用和测试：

```go
type ServerConfig struct {
    Host               string `yaml:"Host" comment:"服务监听地址"`
    Port               int    `yaml:"Port" comment:"服务端口"`
    SessionTTL         int    `yaml:"SessionTTL" comment:"会话密钥过期时间（秒）"`
    InsecureSkipVerify bool   `yaml:"InsecureSkipVerify" comment:"跳过SSH主机密钥验证"`
}

type RedisConfig struct {
    Addr     string `yaml:"Addr" comment:"Redis地址"`
    Password string `yaml:"Password" comment:"Redis密码"`
    DB       int    `yaml:"DB" comment:"Redis数据库编号"`
}

type ESConfig struct {
    Url      string `yaml:"Url" comment:"ES地址"`
    Username string `yaml:"Username" comment:"ES用户名"`
    Password string `yaml:"Password" comment:"ES密码"`
}

type AuditConfig struct {
    LoginAuditIndex  string `yaml:"LoginAuditIndex" comment:"登录审计索引前缀"`
    RecordAuditIndex string `yaml:"RecordAuditIndex" comment:"操作审计索引前缀"`
}

type S3Config struct {
    Endpoint        string `yaml:"Endpoint" comment:"MinIO地址"`
    AccessKeyID     string `yaml:"AccessKeyID" comment:"AccessKey"`
    SecretAccessKey string `yaml:"SecretAccessKey" comment:"SecretKey"`
    UseSSL          bool   `yaml:"UseSSL" comment:"是否使用HTTPS"`
    Bucket          string `yaml:"Bucket" comment:"桶名"`
}

type Config struct {
    Server        ServerConfig `yaml:"Server"`
    Redis         RedisConfig  `yaml:"Redis"`
    ElasticSearch ESConfig     `yaml:"ElasticSearch"`
    Audit         AuditConfig  `yaml:"Audit"`
    S3            S3Config     `yaml:"S3"`
}
```

### 1.2 字段重命名

| 原名 | 新名 | 原因 |
|------|------|------|
| `Server.Ip` | `Server.Host` | `0.0.0.0` 不是严格意义上的 IP，Host 更准确 |
| `S3.EndPoint` | `S3.Endpoint` | Go 惯例小写 p |
| `json:"add"` (Redis) | 删除，仅用 yaml tag | JSON tag 是 typo |
| `commnet:"redis密码"` | `comment:"Redis密码"` | typo 修正 |

### 1.3 config.yaml 同步更新

```yaml
Server:
  Host: 0.0.0.0
  Port: 8000
  SessionTTL: 86400
  InsecureSkipVerify: true
Redis:
  Addr: 127.0.0.1:6379
  Password: ""
  DB: 0
ElasticSearch:
  Url: http://127.0.0.1:9200
  Username: ""
  Password: ""
Audit:
  LoginAuditIndex: gwebssh-login
  RecordAuditIndex: gwebssh-record
S3:
  Endpoint: 127.0.0.1:9000
  AccessKeyID: xxx
  SecretAccessKey: xxx
  UseSSL: false
  Bucket: gwebssh
```

### 1.4 引用更新

需要 grep 替换所有引用：
- `config.Conf.Server.Ip` → `config.Conf.Server.Host`
- `config.Conf.S3.EndPoint` → `config.Conf.S3.Endpoint`

影响文件：`cmd/root.go`、`app/ws/view/view.go`（如有引用）

---

## 2. 配置安全与校验

### 2a. Init() 不再打印密码

**文件:** `config/config.go`

删除 `logger.Info(fmt.Sprintf("解析配置文件：%v", *Conf))`，替换为：
```go
logger.Info("配置文件加载完成")
```

### 2b. 支持环境变量覆盖

在 `Init()` 中添加：
```go
viper.SetEnvPrefix("GWEBSSH")
viper.AutomaticEnv()
```

环境变量格式：`GWEBSSH_SERVER_PORT=9000` 会覆盖 yaml 中的值。

### 2c. 必填校验

**文件:** `config/config.go`

新增 `Validate()` 函数：
```go
func Validate() error {
    if Conf.Server.Port <= 0 || Conf.Server.Port > 65535 {
        return fmt.Errorf("Server.Port 必须在 1-65535 之间")
    }
    if Conf.Redis.Addr == "" {
        return fmt.Errorf("Redis.Addr 不能为空")
    }
    return nil
}
```

**文件:** `cmd/root.go`

在 `Run()` 中调用，校验失败 fatal 退出。

### 2d. 清理 copyright 占位符

**文件:** `cmd/root.go`

删除 `Copyright (C) 2024 NAME HERE <EMAIL ADDRESS>` 行。

---

## 3. 清理死代码和文件命名

### 3a. 删除空文件

- 删除 `app/ws/params/params.go`（空文件，无引用）

### 3b. 文件名修正

- `pkg/middleware/cores.go` → 重命名为 `cors.go`（通过 git mv）

### 3c. 删除死代码

| 位置 | 死代码 |
|------|--------|
| `app/ws/utils/loginAudit/audit.go:133` | `if offset == 0 { offset = 0 }` |
| `pkg/redis/redis.go` | `Exist()` 函数（未被调用） |
| `pkg/asciinema/asciinema.go` | `WriteSize()` 函数（未被调用） |
| `app/api/view/view.go:60` | `fmt.Println(info)` — 调试代码，泄露密码 |

### 3d. import 别名修正

**文件:** `router/router.go`

- `api_view` → `apiview`
- `ws_view` → `wsview`

### 3e. 变量命名修正

| 文件 | 原名 | 新名 |
|------|------|------|
| `pkg/redis/redis.go` | `value_` | `value` |
| `pkg/s3/s3.go` | `BucketName`（局部变量） | `bucketName` |

---

## 4. 统一包模式

### 4a. 审计包合并公共逻辑

**新建文件:** `app/ws/utils/esAudit/base.go`

提取 loginAudit 和 recordAudit 中重复的 WriteData 逻辑：

```go
package esAudit

type Base struct {
    Index    string
    Mappings string
}

func (b *Base) WriteData(data map[string]any) {
    if !es.IsExistsIndex(b.Index) {
        if err := es.CreateIndex(b.Index); err != nil {
            logger.Error(fmt.Sprintf("创建索引失败-%s", err.Error()))
            return
        }
        if err := es.CreateMap(b.Index, b.Mappings); err != nil {
            logger.Error(fmt.Sprintf("创建mapping失败-%s", err.Error()))
            return
        }
    }
    if err := es.InsertData(b.Index, data); err != nil {
        logger.Error(fmt.Sprintf("插入数据失败-%s", err.Error()))
    }
}
```

`loginAudit` 和 `recordAudit` 改为嵌入 `Base`，各自只定义索引名和 mapping。

### 4b. 基础设施初始化统一

`redis.Init()`, `es.Init()`, `s3.Init()` 统一错误处理模式：
- 成功 → 设置全局客户端
- 失败 → `logger.Error` + 设 nil（不阻塞启动，由调用方检查 nil）

当前已基本一致，无需额外修改。

---

## 修改文件清单

| 文件 | 改动类型 |
|------|----------|
| `config/config.go` | 重构结构体、字段重命名、加 Validate()、加环境变量支持 |
| `config/config.yaml` | 字段名同步、注释更新 |
| `cmd/root.go` | 调用 Validate()、删除 copyright、更新引用 |
| `app/ws/view/view.go` | 更新 config 引用（如有） |
| `app/api/view/view.go` | 删除 fmt.Println、更新 config 引用 |
| `router/router.go` | import 别名修正 |
| `pkg/middleware/cores.go` → `cors.go` | 文件重命名 |
| `pkg/redis/redis.go` | 删除 Exist()、修正 value_ 命名 |
| `pkg/asciinema/asciinema.go` | 删除 WriteSize() |
| `pkg/s3/s3.go` | BucketName → bucketName、EndPoint → Endpoint |
| `app/ws/utils/loginAudit/audit.go` | 嵌入 Base、删除死代码 |
| `app/ws/utils/recordAudit/record.go` | 嵌入 Base |
| `app/ws/utils/esAudit/base.go` | **新建** — 公共审计逻辑 |
| `app/ws/params/params.go` | **删除** — 空文件 |

## Verification

```bash
cd backend && go build ./...
```

无编译错误即可。无集成测试可运行。
