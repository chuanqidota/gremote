package loginAudit

import (
	"encoding/json"
	"fmt"
	"time"
	"gremote/app/api/params"
	"gremote/app/ws/utils/esAudit"
	"gremote/config"
	"gremote/pkg/es"
)

type EsAudit struct {
	esAudit.Base
}

func NewEsAudit() *EsAudit {
	return &EsAudit{
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

func (e *EsAudit) UpdateEndTime(keyValue string) {
	es.UpdateByField(e.Index, "key", keyValue, "endTime", time.Now().Format("2006-01-02 15:04:05"))
}

func (e *EsAudit) ReadData(data params.LoginAuditQuery) ([]map[string]any, int64) {
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
	if protocol != "" {
		must = append(must, map[string]any{
			"term": map[string]string{"protocol": protocol},
		})
	} else {
		// When no protocol filter, exclude RDP sessions (show SSH only)
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
	result, count := es.Search(e.Index, string(queryB))
	return result, count
}
