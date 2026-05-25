package params

// Info 登录信息
type Info struct {
	User     string `json:"user" comment:"用户"`
	Source   string `json:"source" comment:"源地址"`
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
	Key  string `form:"key" comment:"redis中的key" binding:"required"`
	Path string `form:"path" comment:"目标服务器上的路径" binding:"required"`
}

// DownLoadFileBody 下载文件参数
type DownLoadFileBody struct {
	Key      string `form:"key" comment:"redis中的key" binding:"required"`
	Path     string `form:"path" comment:"目录地址" binding:"required"`
	FileName string `form:"filename" comment:"文件名" binding:"required"`
}

type LoginAuditQuery struct {
	Offset    int    `form:"offset" comment:"分页-offset"`
	Limit     int    `form:"limit" comment:"分页-limit"`
	User      string `form:"user" comment:"用户信息"`
	Source    string `form:"source" comment:"源地址"`
	Target    string `form:"target" comment:"目的地址"`
	StartTime string `form:"startTime" comment:"开始时间"`
	EndTime   string `form:"endTime" comment:"结束时间"`
	Search    string `form:"search" comment:"搜索"`
	Protocol  string `form:"protocol" comment:"协议类型"`
}

// RDPInfo RDP登录信息
type RDPInfo struct {
	User     string `json:"user" comment:"用户"`
	Source   string `json:"source" comment:"源地址"`
	Target   string `json:"target" comment:"目标地址" binding:"required"`
	Port     int    `json:"port" comment:"端口" binding:"required"`
	Username string `json:"username" comment:"用户名" binding:"required"`
	Password string `json:"password" comment:"密码" binding:"required"`
	Domain   string `json:"domain" comment:"域名"`
}
