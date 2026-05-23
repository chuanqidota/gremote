package esAudit

import (
	"fmt"
	"gwebssh/pkg/es"
	"gwebssh/pkg/logger"
)

type Base struct {
	Index    string
	Mappings string
}

func (b *Base) WriteData(data map[string]any) {
	logger.Info(fmt.Sprintf("存es的日志数据-%v", data))
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
