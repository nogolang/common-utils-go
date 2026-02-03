package dataUtils

// 数组分页
func ArrayPage[K any](array []K, page int, pageSize int) []K {
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(array) {
		return []K{}
	}
	//如果最后一页超过总大小的，那么只取剩余的
	if end > len(array) {
		end = len(array)
	}
	return array[start:end]
}
