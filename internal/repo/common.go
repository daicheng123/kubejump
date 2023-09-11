package repo

import (
	"fmt"
	"github.com/daicheng123/kubejump/pkg/utils"
	"gorm.io/gorm"
	"strings"
)

//type Option interface {
//	apply(db *gorm.DB)
//}

//type OptionFunc func(db *gorm.DB) *gorm.DB

//func (fn OptionFunc) apply(db *gorm.DB) {
//	fn(db)
//	//optionFunc()
//}

func IsActive(active bool) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if !active {
			return db
		}
		return db.Where("activate = ?", active)
	}
}

func OrderBy(sortBy string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(sortBy) <= 0 {
			return db
		}
		return db.Order(sortBy)
	}
}

func Paginate(pageSize int, pageNo int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if pageNo <= 0 {
			pageNo = 1
		}
		offset := (pageNo - 1) * pageSize
		//offset := pageNo
		return db.Offset(offset).Limit(pageSize)
	}
}

func SearchBy(fields map[string]interface{}) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		search := ""
		for k, v := range fields {
			if !utils.IsZero(v) {
				search += fmt.Sprintf("%s like '%%%v%%' AND ", k, v)
			}
		}
		search = strings.TrimSuffix(search, " AND ")
		return db.Where(search)
	}
}

func SearchPodBy(search string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(search) == 0 {
			return db
		}

		var sql string
		for _, field := range []string{"cluster_ref", "pod_name", "namespace", "pod_ip"} {
			sql += fmt.Sprintf("%s like '%%%v%%' or ", field, search)
		}
		if num, err := validConvertNum(search); err == nil {
			sql += fmt.Sprintf("%s like '%%%v%%' or ", "id", num)
		}
		sql = strings.TrimSuffix(sql, " or ")
		return db.Where(sql)
	}
}

func PaginatePods(pageSize int, offset int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		//if offset <= 0 {
		//	pageNo = 1
		//}
		if pageSize <= 0 {
			pageSize = 10
		}
		//offset := (pageNo - 1) * pageSize
		//offset := pageNo
		return db.Offset(offset).Limit(pageSize)
	}
}
