package view

import (
	"fmt"
	"time"
	"webssh-go/app/api/params"
	"webssh-go/pkg/response"

	"strings"
	"webssh-go/pkg/redis"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type apiHandle struct {
}

var ApiHandle = new(apiHandle)

// ObtainKey 获取key
func (a *apiHandle) ObtainKey(c *gin.Context) {
	var info params.Info
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Fail(c, fmt.Sprintf("参数错误-%s", err.Error()))
		return
	}
	_uuid := uuid.New().String()
	uuid_ := strings.Replace(_uuid, "-", "", -1)
	fmt.Println(uuid_)

	if err := redis.Set(uuid_, info, time.Second*60); err != nil {
		response.Fail(c, fmt.Sprintf("redis设置失败-%s", err.Error()))
		return
	}
	response.Success(c, "执行成功", map[string]string{"result": uuid_})
}

