package gormUtils

import "gorm.io/gorm"

func Pagination(page int, pageSize int) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		//第10w页就不分了，不然会有深分页的问题
		if page <= 0 {
			page = 1
		} else if page >= 100000 {
			page = 100000
		}

		//如果是-1，那么就是查询所有，我们接收前台参数的时候，必须要去校验大于0
		//这个-1一般只用于给内部方法使用过
		if pageSize == -1 {
			return db
		}

		if pageSize <= 0 {
			pageSize = 10
		} else if pageSize >= 50 {
			pageSize = 50
		}

		//这里要减1，因为第1页数据从0开始
		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}
