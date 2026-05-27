package es

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gremote/config"
	"gremote/pkg/logger"

	"github.com/olivere/elastic/v7"
)

var ElasticSearch *elastic.Client

// IsReady 检查ES客户端是否可用
func IsReady() bool {
	return ElasticSearch != nil
}

func Init() {
	client, err := elastic.NewClient(
		elastic.SetURL(config.Conf.ElasticSearch.Url),
		elastic.SetBasicAuth(config.Conf.ElasticSearch.Username, config.Conf.ElasticSearch.Password), // 用户名和密码
		elastic.SetSniff(false), // 用于关闭 Sniff 不然会出现no active connection found: no Elasticsearch node available
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
	if !IsReady() {
		return errors.New("elasticsearch not initialized")
	}
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
	if !IsReady() {
		return errors.New("elasticsearch not initialized")
	}
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
	if !IsReady() {
		return false
	}
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
	if !IsReady() {
		return errors.New("elasticsearch not initialized")
	}
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
	if !IsReady() {
		return errors.New("elasticsearch not initialized")
	}
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
	if !IsReady() {
		return nil, 0
	}
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

// UpdateByField 根据字段查询并更新指定字段的值
func UpdateByField(index, byField, ByValue, ChField, ChValue string) {
	if !IsReady() {
		return
	}
	query := elastic.NewTermQuery(byField, ByValue)
	script := elastic.NewScript("ctx._source."+ChField+" = '"+ChValue+"'")
	_, err := ElasticSearch.UpdateByQuery().Index(index).Query(query).Script(script).Do(context.Background())
	if err != nil {
		logger.Error(fmt.Sprintf("更新出错-%s", err.Error()))
	}
}
