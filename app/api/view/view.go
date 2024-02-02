package view

import (
	"fmt"
	"webssh-go/app/api/params"
	"webssh-go/pkg/logger"
	"webssh-go/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"strings"
)

type apiHandle struct {
}

var ApiHandle = new(apiHandle)

// ObtainKey 获取key
func (a *apiHandle) ObtainKey(c *gin.Context) {
	var info params.Info
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Fail(c,fmt.Sprintf("参数错误-%s",err.Error()))
		return
	}
	_uuid := uuid.New().String()
	uuid_ := strings.Replace(_uuid, "-", "", -1)
	fmt.Sprintln(uuid_)
	
	return
}
