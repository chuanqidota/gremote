package loginAudit

import (
	"encoding/json"
	"fmt"
	"time"
	"webssh-go/app/api/params"
	"webssh-go/config"
	"webssh-go/pkg/es"
	"webssh-go/pkg/logger"
)

type EsAudit struct {
	Index string `json:"index" comment:"索引"`
}

// NewEsAudit 实例化
func NewEsAudit() *EsAudit {
	return &EsAudit{
		Index: fmt.Sprintf("%s-%s", config.Conf.Audit.LoginAuditIndex, time.Now().Format("2006-01")),
	}
}

// WriteData 写入数据
func (e *EsAudit) WriteData(data map[string]any) {
	logger.Info(fmt.Sprintf("存es的日志数据-%v", data))
	index := e.Index
	if !es.IsExistsIndex(index) {
		if err := es.CreateIndex(index); err != nil {
			logger.Error(fmt.Sprintf("创建索引失败-%s", err.Error()))
			return
		} else {
			mappings := `{
							"properties":{
									"key":{
										"type":"keyword"
									},
									"startTime":{
										"type":"keyword"
									},
									"endTime": {
										"type":"keyword"
									},
									"user":{
										"type":"keyword"
									},
									"source": {
										"type":"keyword"
									},
									"target": {
										"type":"keyword"
									}
							}
						}`
			if err = es.CreateMap(index, mappings); err != nil {
				logger.Error(fmt.Sprintf("创建mappings失败-%s", err.Error()))
				return
			}
		}
	}
	if err := es.InsertData(index, data); err != nil {
		logger.Error(fmt.Sprintf("插入数据失败-%s", err.Error()))
		return
	}
}

func (e *EsAudit) UpdateEndTime(keyValue string) {
	es.UpdateByField(e.Index, "key", keyValue, "endTime", time.Now().Format("2006-01-02 15:04:05"))
}

// ReadData 读取数据
func (e *EsAudit) ReadData(data params.LoginAuditQuery) ([]map[string]any, int64) {
	user := data.User
	source := data.Source
	target := data.Target
	startTime := data.StartTime
	endTime := data.EndTime
	search := data.Search
	limit := data.Limit
	offset := data.Offset

	var must []map[string]any
	if user != "" {
		must = append(must, map[string]any{
			"wildcard": map[string]string{
				"user": fmt.Sprintf("*%s*", user),
			},
		})
	}
	if source != "" {
		must = append(must, map[string]any{
			"wildcard": map[string]string{
				"source": fmt.Sprintf("*%s*", source),
			},
		})
	}
	if target != "" {
		must = append(must, map[string]any{
			"wildcard": map[string]string{
				"target": fmt.Sprintf("*%s*", target),
			},
		})
	}
	if startTime != "" && endTime != "" {
		must = append(must, map[string]any{
			"range": map[string]any{
				"timestamp": map[string]string{
					"gte": startTime,
					"lte": endTime,
				},
			},
		})
	}

	var should []map[string]any
	if search != "" {
		should = append(should, map[string]any{
			"wildcard": map[string]string{
				"user": fmt.Sprintf("*%s*", search),
			},
		}, map[string]any{
			"wildcard": map[string]string{
				"source": fmt.Sprintf("*%s*", search),
			},
		}, map[string]any{
			"wildcard": map[string]string{
				"target": fmt.Sprintf("*%s*", search),
			},
		})
	}

	if offset == 0 {
		offset = 0
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
	}
	queryB, ok := json.Marshal(query)
	if ok != nil {
		return nil, 0
	}
	result, count := es.Search(e.Index, string(queryB))
	return result, count
}
