package repo

import (
	"context"
	"errors"
	"fmt"
	"github.com/daicheng123/kubejump/internal/base/data"
	"github.com/daicheng123/kubejump/internal/entity"
	"gorm.io/gorm"
	"sync"
)

type ClusterRepo struct {
	lock sync.Mutex
	data *data.Data
}

func (cr *ClusterRepo) ListClustersByStatus(isActive bool) ([]*entity.ClusterConfig, error) {

	filter := &entity.ClusterConfig{
		Activate: isActive,
	}

	var clusterList = make([]*entity.ClusterConfig, 0)
	db := cr.data.DB.Session(&gorm.Session{}).Where(filter).Find(&clusterList)
	return clusterList, db.Error
}

func (cr *ClusterRepo) GetClustersInfo(filter, result *entity.ClusterConfig) error {
	if filter == nil {
		return errors.New("filter is Nil")
	}

	return cr.data.DB.Session(&gorm.Session{}).Where(filter).First(result).Error
}

func (cr *ClusterRepo) UpdateCluster(_ context.Context, cluster *entity.ClusterConfig) error {
	return cr.data.DB.Session(&gorm.Session{}).Updates(cluster).Error

}

func (cr *ClusterRepo) CreateCluster(_ context.Context, cluster *entity.ClusterConfig) error {
	db := cr.data.DB.Session(&gorm.Session{}).Create(cluster)
	fmt.Println(db.Error, "hello world")
	return db.Error
}
func NewClusterRepo() entity.ClusterRepo {
	return &ClusterRepo{
		data: data.DefaultData,
	}

}
