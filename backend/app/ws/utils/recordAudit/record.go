package recordAudit

import (
	"encoding/json"
	"fmt"
	"time"
	"gremote/app/ws/utils/esAudit"
	"gremote/config"
	"gremote/pkg/es"
)

// EsRecord 操作录制审计，继承 ES 基础写入能力
type EsRecord struct {
	esAudit.Base
}

// NewEsRecord 创建操作录制审计实例，索引按月分区
func NewEsRecord() *EsRecord {
	return &EsRecord{
		Base: esAudit.Base{
			Index: fmt.Sprintf("%s-%s", config.Conf.Audit.RecordAuditIndex, time.Now().Format("2006-01")),
			Mappings: `{
				"properties":{
					"key":{"type":"keyword"},
					"timeStamp":{"type":"keyword"},
					"history":{"type":"keyword"}
				}
			}`,
		},
	}
}

// ReadData 按会话 key 分页读取所有录制事件，按时间戳升序排列
func (e *EsRecord) ReadData(key string) []map[string]any {
	result := make([]map[string]any, 0)
	pageNum := 1
	pageSize := 10000
	for {
		from := (pageNum - 1) * pageSize
		query := map[string]any{
			"query": map[string]any{
				"bool": map[string]any{
					"must": []map[string]any{
						{"match": map[string]string{"key": key}},
					},
				},
			},
			"sort": []map[string]any{
				{"timeStamp": map[string]string{"order": "asc"}},
			},
			"from": from,
			"size": pageSize,
		}
		queryB, err := json.Marshal(query)
		if err != nil {
			return result
		}
		res, _ := es.Search(e.Index, string(queryB))
		if len(res) == 0 {
			break
		}
		result = append(result, res...)
		pageNum++
	}
	return result
}
