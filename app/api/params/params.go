package params

// Info 登录信息
type Info struct {
	User     string `json:"user" comment:"用户" binding:"required"`
	Source   string `json:"source" comment:"源地址" binding:"required"`
	Target   string `json:"target" comment:"目标地址" binding:"required"`
	Username string `json:"username" comment:"主机用户名" binding:"required"`
	Password string `json:"password" comment:"主机密码" binding:"required"`
	Port     int    `json:"port" comment:"端口" binding:"required"`
}

// ListFileBody 文件列表参数
type ListFileBody struct {
	Key  string `form:"key" comment:"redis中的key" binding:"required"`
	Path string `form:"path" comment:"目标服务器上的路径" binding:"required"`
}

// UploadFileBody 上传文件参数
type UploadFileBody struct {
	Key  string `json:"key" comment:"redis中的key" binding:"required"`
	Path string `json:"path" comment:"目标服务器上的路径" binding:"required"`
}

// DownLoadFileBody 下载文件参数
type DownLoadFileBody struct {
	Key      string `form:"key" comment:"redis中的key" binding:"required"`
	Path     string `form:"path" comment:"目录地址" binding:"required"`
	FileName string `form:"filename" comment:"文件名" binding:"required"`
}
