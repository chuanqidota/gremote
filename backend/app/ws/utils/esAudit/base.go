package esAudit

import (
	"fmt"
	"gremote/pkg/es"
	"gremote/pkg/logger"
)

// Base ES 审计写入基础结构，提供索引名和 mapping 定义
type Base struct {
	Index    string // ES 索引名（按月分区，如 gremote-login-2024-01）
	Mappings string // ES 索引 mapping JSON
}

// WriteData 将审计数据写入 ES，索引不存在时自动创建
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
