package es

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"webssh-go/config"
	"webssh-go/pkg/logger"

	"github.com/olivere/elastic/v7"
)

var ElasticSearch *elastic.Client

func Init() {
	client, err := elastic.NewClient(
		elastic.SetURL(config.Conf.ElasticSearch.Url),
		elastic.SetBasicAuth(config.Conf.ElasticSearch.Username, config.Conf.ElasticSearch.Password), // 用户名和密码
	)
	if err != nil {
		logger.Error(fmt.Sprintf("es连接失败-%s", err.Error()))
		return
	}
	ElasticSearch = client
	logger.Info(fmt.Sprintf("es连接成功"))
}

// CreateIndex
//
//	@Description: 创建索引
//	@param index
//	@return error
func CreateIndex(index string) error {
	createIndex, err := ElasticSearch.CreateIndex(index).Do(context.Background())
	if err != nil || !createIndex.Acknowledged {
		return err
	} else {
		return nil
	}
}

// DeleteIndex
//
//	@Description: 删除索引
//	@param index
//	@return error
func DeleteIndex(index string) error {
	deleteIndex, err := ElasticSearch.DeleteIndex(index).Do(context.Background())
	if err != nil || !deleteIndex.Acknowledged {
		return err
	} else {
		return nil
	}
}

// IsExistsIndex
//
//	@Description: 是否存在索引
//	@param index
//	@return bool
func IsExistsIndex(index string) bool {
	exists, err := ElasticSearch.IndexExists(index).Do(context.Background())
	if err != nil {
		return false
	} else {
		return exists
	}
}

// CreateMap
//
//	@Description: 创建索引
//	@param index
//	@param mappings
//
/*
mappings := `{
			"properties":{
				"title":{
					"type":"keyword"
				},
				"age":{
					"type":"keyword"
				}
			}
	}`
*/
//		@return error
func CreateMap(index string, mappings string) error {
	do, err := ElasticSearch.PutMapping().Index(index).BodyString(mappings).Do(context.Background())
	if err != nil || !do.Acknowledged {
		return err
	} else {
		return nil
	}
}

//
// InsertData
//  @Description: 插入数据
//  @param index
//  @param data
/*
data := map[string]interface{}{
		"title":   "Sample Document",
		"content": "This is a sample document for Elasticsearch indexing.",
		"age":     25,
		"category": "tech",
	}
*/
//  @return error
//
func InsertData(index string, data map[string]any) error {
	resp, err := ElasticSearch.Index().
		Index(index).
		BodyJson(data).
		Do(context.Background())
	if err != nil {
		return err
	}
	if resp.Result != "created" {
		return errors.New("创建出错")
	}
	return nil
}

// Search
//
//	@Description: 查询
//	@param index
//	@param query
/*
query := `{
		"query":{
			"bool":{
				"must":[
					{
						"wildcard":{ // 通用符匹配
							"title":"标题*"
						}
					}
				]
			}
		},
		"from":0, // 分页
		"size":10
	}`
*/
// @return []*elastic.SearchHit
// @return error
func Search(index string, query string) (result []map[string]any, count int64) {
	searchResult, err := ElasticSearch.Search().
		Index(index).
		Source(query).
		Do(context.Background())
	if err != nil {
		return nil, 0
	}
	if searchResult.Hits.TotalHits.Value > 0 {
		for _, hit := range searchResult.Hits.Hits {
			//return hit.Source
			var item map[string]any
			_ = json.Unmarshal(hit.Source, &item)
			result = append(result, item)
		}
		return result, searchResult.Hits.TotalHits.Value
	} else {
		return nil, 0
	}
}

// UpdateByField 根据字段进行更新
func UpdateByField(index, byField, ByValue, ChField, ChValue string) {
	// 构建查询条件
	termQuery := elastic.NewTermQuery(byField, ByValue)
	// 执行查询
	searchResult, err := ElasticSearch.Search().
		Index(index).
		Query(termQuery).
		Do(context.Background())
	if err != nil {
		logger.Error(fmt.Sprintf("查询出错-%s", err.Error()))
		return
	}
	for _, hit := range searchResult.Hits.Hits {
		// 提取文档 ID
		docID := hit.Id

		// 创建更新请求
		updateRequest := elastic.NewBulkUpdateRequest().
			Index(index).
			Id(docID).
			Doc(map[string]interface{}{ChField: ChValue})

		// 将更新请求添加到批量操作中
		_, err := ElasticSearch.Bulk().Add(updateRequest).Do(context.Background())
		if err != nil {
			// 处理错误
			logger.Error(fmt.Sprintf("更新出错-%s", err.Error()))
		}
	}
}
