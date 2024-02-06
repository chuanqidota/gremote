package params

// ItemInfo 目标机器信息
type ItemInfo struct {
	Target   string `json:"addr" comment:"目标地址" binding:"required"`
	Username string `json:"username" comment:"主机用户名" binding:"required"`
	Password string `json:"password" comment:"主机密码" binding:"required"`
	Port     int    `json:"port" comment:"端口" binding:"required"`
}

// Info 登录信息
type Info struct {
	User     string `json:"user" comment:"用户" binding:"required"`
	Source   string `json:"source" comment:"源地址" binding:"required"`
	ItemInfo ItemInfo
}

// ListFileBody 文件列表参数
type ListFileBody struct {
	Key  string `form:"key" comment:"redis中的key" binding:"required"`
	Path string `form:"path" comment:"目标服务器上的路径" binding:"required"`
}


type UploadFileBody struct {
	Key  string `form:"key" comment:"redis中的key" binding:"required"`
	Path string `form:"path" comment:"目标服务器上的路径" binding:"required"`
}