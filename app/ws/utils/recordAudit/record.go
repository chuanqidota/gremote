package recordAudit

import (
	"encoding/json"
	"fmt"
	"time"
	"webssh-go/config"
	"webssh-go/pkg/es"
	"webssh-go/pkg/logger"
)

type EsRecord struct {
	Index string `json:"index" comment:"索引"`
}

func NewEsRecord() *EsRecord {
	return &EsRecord{
		Index: fmt.Sprintf("%s-%s", config.Conf.Audit.RecordAuditIndex, time.Now().Format("2006-01")),
	}
}

// WriteData 写入操作记录到es中
func (e *EsRecord) WriteData(data map[string]any) {
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
									"timeStamp":{
										"type":"keyword"
									},
									"history":{
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

// ReadData 从es中读取记录
func (e *EsRecord) ReadData(key string) []map[string]any {
	result := make([]map[string]any, 0)
	index := e.Index
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
		res, _ := es.Search(index, string(queryB))
		if len(res) == 0 {
			break
		}
		result = append(result, res...)
		pageNum++
	}
	return result
}
