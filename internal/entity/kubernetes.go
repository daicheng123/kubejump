package entity

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/util/json"
	"reflect"
	"strings"
)

const (
	httpPrefix = "http://"
)

type ClusterRepo interface {
	ListClustersByStatus(ctx context.Context, isActive bool) ([]*ClusterConfig, error)
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
	c.masterUrl()
	return fmt.Sprintf("%s_%s_%d", c.ClusterName, c.MasterUrl, c.ConfigVersion)
}

func (c *ClusterConfig) OldClientUniqKey() string {
	c.masterUrl()
	return fmt.Sprintf("%s_%s_%d", c.ClusterName, c.MasterUrl, c.ConfigVersion-1)
}

func (c *ClusterConfig) masterUrl() {
	if strings.HasPrefix(c.MasterUrl, httpPrefix) {
		ss := strings.Split(c.MasterUrl, "//")
		if len(ss) >= 2 {
			c.MasterUrl = strings.Join(ss[1:], "")
		}
	}
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

type NamespaceRepo interface {
	CreateOrUpdateNS(_ context.Context, ns *Namespace) error
	DeleteNSByName(_ context.Context, name, uniqKey string) error
}
type Namespace struct {
	BaseModel
	NamespaceName  string `gorm:"not null;type:varchar(128);uniqueIndex:idx_namespace_cluster_uniq_key"`
	ClusterUniqKey string `gorm:"type:varchar(256);not null;uniqueIndex:idx_namespace_cluster_uniq_key"`
	ResourceKind   string `gorm:"-"`
}

func (c *Namespace) TableName() string {
	return "namespace_info"
}

type PodRepo interface {
	CreateOrUpdatePod(_ context.Context, pod *Pod) error
	DeletePodByNameAndNamespace(_ context.Context, name, ns string, id uint) error
	ListPodsWithPreLoadCluster(_ context.Context, filter *Pod, sortBy string) ([]*Pod, error)
	PreloadPods(_ context.Context, filter *Pod, reqParam *PaginationParam) ([]*Pod, error)
	CountPods(_ context.Context, filter *Pod, reqParam *PaginationParam) (int, error)
}

type Pod struct {
	BaseModel
	PodName      string         `gorm:"not null;type:varchar(256);uniqueIndex:idx_namespace_pod_name_cluster_ref"`
	Namespace    string         `gorm:"not null;type:varchar(256);uniqueIndex:idx_namespace_pod_name_cluster_ref"`
	PodIP        string         `gorm:"pod_ip;varchar(15)"`
	Status       string         `gorm:"type:varchar(28);not null"`
	ClusterRef   uint           `gorm:"not null;uniqueIndex:idx_namespace_pod_name_cluster_ref"`
	Cluster      *ClusterConfig `gorm:"foreignKey:ClusterRef"`
	Containers   []*Container   `gorm:"foreignKey:PodRef;"`
	ResourceKind string         `gorm:"-"`
}

func (c *Pod) TableName() string {
	return "pod_info"
}

type Container struct {
	BaseModel
	ContainerName string `gorm:"not null;type:varchar(256)"`
	ContainerIP   string `gorm:"not null;type:varchar(256)"`
	Status        string `gorm:"not null;type:varchar(28)"`
	PodRef        uint
}

func (c *Container) TableName() string {
	return "container_info"
}
