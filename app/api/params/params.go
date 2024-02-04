package params

type ItemInfo struct {
	Username string `json:"username" comment:"主机用户名" binding:"required"`
	Password string `json:"password" comment:"主机密码" binding:"required"`
	Port     int    `json:"port" comment:"端口" binding:"required"`
	User     string `json:"user" comment:"用户" binding:"required"`
}

// Info 登录信息
type Info struct {
	Source   string `json:"source" comment:"源地址"`
	ItemInfo ItemInfo
}

type ListFileBody struct {
	Key  string `json:"key" comment:"redis中的key" binding:"required"`
	Path string `json:"path" comment:"目标服务器上的路径" binding:"required"`
}
