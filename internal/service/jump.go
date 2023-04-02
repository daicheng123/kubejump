package service

import (
	"context"
	"github.com/daicheng123/kubejump/internal/entity"
	"gorm.io/gorm"
)

type JMService struct {
	clusterRepo entity.ClusterRepo
	userRepo    entity.UserRepo
}

func NewJMService(
	clusterRepo entity.ClusterRepo,
	userRepo entity.UserRepo) *JMService {
	return &JMService{
		clusterRepo: clusterRepo,
		userRepo:    userRepo,
	}
}

func (jms *JMService) GetUserById(ctx context.Context, userID int) (user *entity.User, err error) {
	filter := &entity.User{Model: gorm.Model{ID: uint(userID)}}
	return jms.userRepo.GetInfoByID(ctx, filter)
}

func (jms *JMService) GetKubernetesCfg(id int) (result *entity.ClusterConfig, err error) {
	filter := &entity.ClusterConfig{Model: gorm.Model{ID: uint(id)}}
	result = &entity.ClusterConfig{}
	err = jms.clusterRepo.GetClustersInfo(filter, result)
	return
}

func (jms *JMService) ListClusterConfig() ([]*entity.ClusterConfig, error) {
	return jms.clusterRepo.ListClustersByStatus(true)
}

//func(jms *JMService) SyncResourcesToStore() {
//
//}
