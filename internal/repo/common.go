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
