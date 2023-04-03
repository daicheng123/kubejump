package entity

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/util/json"
	"reflect"
)

type ClusterRepo interface {
	ListClustersByStatus(isActive bool) ([]*ClusterConfig, error)
	GetClustersInfo(filter, result *ClusterConfig) error
	CreateCluster(ctx context.Context, cluster *ClusterConfig) error
	UpdateCluster(ctx context.Context, cluster *ClusterConfig) error
	//CreateOrUpdateCluster(ctx context.Context, cluster *ClusterConfig) error
}

type ClusterConfig struct {
	BaseModel
	ClusterName string `json:"cluster_name" gorm:"not null;unique" binding:"required"`
	MasterUrl   string `json:"master_url"   gorm:"not null;" binding:"required"`
	Env         string `json:"env"   gorm:"not null" binding:"required"`
	Activate    bool   `json:"activate" gorm:"type:boolean" binding:"required"` // 0 当前不可用，1 已激活
	InitNode    bool   `json:"init_node" gorm:"type:boolean"`                   // 0 未同步， 1 已同步
	InitPod     bool   `json:"init_pod" gorm:"type:boolean"`                    // 0 未同步， 1 已同步
	//Platform      string `gorm:"not null;unique_index:platform_env"`
	CaData        string `json:"ca_data" gorm:"type:text;" binding:"required"`
	BearerToken   string `json:"bearer_token" gorm:"type:text;not null" binding:"required"`
	LastApply     string `json:"-" gorm:"type:text"`
	ConfigVersion int    `json:"config_version" gorm:"default:0" binding:"required"`
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

func (c *ClusterConfig) Marshal() (s []byte, e error) {
	return json.Marshal(c)
}

func (c *ClusterConfig) Unmarshal(data []byte) (*ClusterConfig, error) {
	err := json.Unmarshal(data, c)
	return c, err
}

func (c *ClusterConfig) IsEmpty() bool {
	return reflect.DeepEqual(c, &ClusterConfig{})
}

type Namespace struct {
	NamespaceName  string
	ClusterUniqKey string
	ResourceKind   string
}

type PodRepo interface {
	AddPod(ctx context.Context, pod *Pod) error
	CreateOrUpdatePod(_ context.Context, pod *Pod) error
}

type Pod struct {
	BaseModel
	PodName        string `gorm:"not null;type:varchar(256);unique_index:idx_namespace_pod_name"`
	Namespace      string `gorm:"not null;type:varchar(256);unique_index:idx_namespace_pod_name"`
	Status         string `gorm:"type:varchar(28);not null"`
	ClusterUniqKey string `gorm:"type:varchar(256);not null"`
	ResourceKind   string `gorm:"-"`
}

func (c *Pod) TableName() string {
	return "pods"
}
