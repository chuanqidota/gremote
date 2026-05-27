package loginAudit

import (
	"encoding/json"
	"fmt"
	"time"
	"gremote/app/api/params"
	"gremote/app/audit/esAudit"
	"gremote/config"
	"gremote/pkg/elasticsearch"
)

// LoginAudit 登录审计，继承 ES 基础写入能力
type LoginAudit struct {
	esAudit.Base
}

// NewLoginAudit 创建登录审计实例，索引按月分区
func NewLoginAudit() *LoginAudit {
	return &LoginAudit{
		Base: esAudit.Base{
			Index: fmt.Sprintf("%s-%s", config.Conf.Audit.LoginAuditIndex, time.Now().Format("2006-01")),
			Mappings: `{
				"properties":{
					"key":{"type":"keyword"},
					"startTime":{"type":"keyword"},
					"endTime":{"type":"keyword"},
					"user":{"type":"keyword"},
					"source":{"type":"keyword"},
					"target":{"type":"keyword"},
					"protocol":{"type":"keyword"}
				}
			}`,
		},
	}
}

// UpdateEndTime 会话结束时更新审计记录的 endTime 字段
func (e *LoginAudit) UpdateEndTime(keyValue string) {
	elasticsearch.UpdateByField(e.Index, "key", keyValue, "endTime", time.Now().Format("2006-01-02 15:04:05"))
}

// ReadData 根据查询条件分页读取登录审计记录，支持按用户/源地址/目标/协议/时间范围筛选
func (e *LoginAudit) ReadData(data params.LoginAuditQuery) ([]map[string]any, int64) {
	user := data.User
	source := data.Source
	target := data.Target
	startTime := data.StartTime
	endTime := data.EndTime
	search := data.Search
	limit := data.Limit
	offset := data.Offset
	protocol := data.Protocol

	var must []map[string]any
	if user != "" {
		must = append(must, map[string]any{
			"wildcard": map[string]string{"user": fmt.Sprintf("*%s*", user)},
		})
	}
	if source != "" {
		must = append(must, map[string]any{
			"wildcard": map[string]string{"source": fmt.Sprintf("*%s*", source)},
		})
	}
	if target != "" {
		must = append(must, map[string]any{
			"wildcard": map[string]string{"target": fmt.Sprintf("*%s*", target)},
		})
	}
	if startTime != "" && endTime != "" {
		must = append(must, map[string]any{
			"range": map[string]any{
				"timestamp": map[string]string{"gte": startTime, "lte": endTime},
			},
		})
	}
	if protocol == "all" {
		// Show all protocols (both SSH and RDP)
	} else if protocol == "rdp" {
		must = append(must, map[string]any{
			"term": map[string]string{"protocol": "rdp"},
		})
	} else if protocol == "ssh" {
		// SSH records have no protocol field, exclude RDP
		must = append(must, map[string]any{
			"bool": map[string]any{
				"must_not": []map[string]any{
					{"term": map[string]string{"protocol": "rdp"}},
				},
			},
		})
	} else {
		// Default: show SSH only (exclude RDP)
		must = append(must, map[string]any{
			"bool": map[string]any{
				"must_not": []map[string]any{
					{"term": map[string]string{"protocol": "rdp"}},
				},
			},
		})
	}

	var should []map[string]any
	if search != "" {
		should = append(should, map[string]any{
			"wildcard": map[string]string{"user": fmt.Sprintf("*%s*", search)},
		}, map[string]any{
			"wildcard": map[string]string{"source": fmt.Sprintf("*%s*", search)},
		}, map[string]any{
			"wildcard": map[string]string{"target": fmt.Sprintf("*%s*", search)},
		})
	}

	if limit == 0 {
		limit = 10
	}

	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must":   must,
				"should": should,
			},
		},
		"from": offset,
		"size": limit,
		"sort": []map[string]any{
			{"startTime": map[string]string{"order": "desc"}},
		},
	}
	queryB, err := json.Marshal(query)
	if err != nil {
		return nil, 0
	}
	result, count := elasticsearch.Search(e.Index, string(queryB))
	return result, count
}
