package params

// Info 登录信息
type Info struct {
	Username string `json:"username" comment:"主机用户名" binding:"required"`
	Password string `json:"password" comment:"主机密码" binding:"required"`
	Port     int    `json:"port" comment:"端口" binding:"required"`
	User     string `json:"user" comment:"用户" binding:"required"`
	Source   string `json:"source" comment:"源地址"`
}
