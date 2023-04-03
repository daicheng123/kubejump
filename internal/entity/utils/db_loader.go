package utils

import (
	"github.com/daicheng123/kubejump/internal/entity"
	"gorm.io/gorm"
)

func InitDBSchema(db *gorm.DB) error {
	return autoMigrate(db)
}

//autoMigrate 自动根据数据模型建表，表名为实体名的蛇形表示
func autoMigrate(db *gorm.DB) (err error) {
	err = db.Set("gorm:table_options", "ENGINE=InnoDB").
		AutoMigrate(
			&entity.ClusterConfig{},
			&entity.User{},
			&entity.Pod{},
		)
	return
}
