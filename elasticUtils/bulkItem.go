package elasticUtils

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
	var allJson string
	for _, item := range *receiver {
		firstMarshal, _ := json.Marshal(item.First)
		bodyMarshal, _ := json.Marshal(item.Body)
		oneItem := string(firstMarshal) + "\n" + string(bodyMarshal)
		allJson += oneItem + "\n"
	}
	return []byte(allJson)
}
