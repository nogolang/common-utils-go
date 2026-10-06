package elasticUtils

import "strings"

import "encoding/json"

// 用于批量插入，es的go typeClient无法批量，因为es的批量操作不是标准json
// 那我们自己写一个结构体拼接json即可
type EsItem struct {
	First map[string]interface{} `json:"index"`
	Body  map[string]interface{} `json:"body"`
}

type EsItems []EsItem

// 把json拼接起来即可
func (receiver *EsItems) GetJson() []byte {
	var allJson strings.Builder
	for _, item := range *receiver {
		firstMarshal, _ := json.Marshal(item.First)
		// bulk 的源行格式跟动作走：index/create 的源行是全量裸文档；
		// update 的源行不是文档而是更新指令体，只认 doc/script 等字段，
		// 所以 Body 必须包一层 doc（部分更新），否则 ES 报
		// [UpdateRequest] unknown field [xxx]，整批 400
		var bodyMarshal []byte
		if _, isUpdate := item.First["update"]; isUpdate {
			bodyMarshal, _ = json.Marshal(map[string]interface{}{"doc": item.Body})
		} else {
			bodyMarshal, _ = json.Marshal(item.Body)
		}
		oneItem := string(firstMarshal) + "\n" + string(bodyMarshal)
		allJson.WriteString(oneItem + "\n")
	}
	return []byte(allJson.String())
}
