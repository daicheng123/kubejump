package entity

import (
	"fmt"
	"gorm.io/gorm"
)

type ClusterRepo interface {
	ListClustersByStatus(isActive bool) ([]*ClusterConfig, error)
	GetClustersInfo(filter, result *ClusterConfig) error
}

type ClusterConfig struct {
	gorm.Model
	ClusterName   string `gorm:"not null;unique"`
	MasterUrl     string `gorm:"not null;"`
	Env           string `gorm:"not null;unique_index:platform_env"`
	Activate      bool   `gorm:"type:boolean"` // 0 当前不可用，1 已激活
	InitNode      bool   `gorm:"type:boolean"` // 0 未同步， 1 已同步
	InitPod       bool   `gorm:"type:boolean"` // 0 未同步， 1 已同步
	Platform      string `gorm:"not null;unique_index:platform_env"`
	CaData        string `gorm:"type:text;"`
	BearerToken   string `gorm:"type:text;not null"`
	ConfigVersion int    `gorm:"default:0"`
}

func (c *ClusterConfig) ClientUniqKey() string {
	return fmt.Sprintf("%s_%s_%d", c.ClusterName, c.MasterUrl, c.ConfigVersion)
}

func (c *ClusterConfig) OldClientUniqKey() string {
	return fmt.Sprintf("%s_%s_%d", c.ClusterName, c.MasterUrl, c.ConfigVersion-1)
}

func (c *ClusterConfig) TableName() string {
	return "clusters"
}

type Namespace struct {
	NamespaceName  string
	ClusterUniqKey string
	ResourceKind   string
}

type Pod struct {
	gorm.Model
	//PodNamespace string
	PodName        string
	Status         bool
	ClusterUniqKey string
	ResourceKind   string
}
