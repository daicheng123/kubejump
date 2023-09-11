package entity

import (
	"gorm.io/gorm"
	"time"
)

type BaseModel struct {
	ID        uint `gorm:"primarykey" json:"id,omitempty"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type UniqKeyGenerator interface {
	ClientUniqKey() string
}
