package service

import (
	"context"
	"github.com/daicheng123/kubejump/internal/entity"
	jsonpatch "github.com/evanphx/json-patch"
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
	filter := &entity.User{BaseModel: entity.BaseModel{ID: uint(userID)}}
	return jms.userRepo.GetInfoByID(ctx, filter)
}

func (jms *JMService) GetKubernetesCfg(id int) (result *entity.ClusterConfig, err error) {
	filter := &entity.ClusterConfig{BaseModel: entity.BaseModel{ID: uint(id)}}
	result = &entity.ClusterConfig{}
	err = jms.clusterRepo.GetClustersInfo(filter, result)
	return
}

func (jms *JMService) ListClusterConfig() ([]*entity.ClusterConfig, error) {
	return jms.clusterRepo.ListClustersByStatus(true)
}

// ApplyK8sCluster create or update kubernetes cluster
func (jms *JMService) ApplyK8sCluster(ctx context.Context, cluster *entity.ClusterConfig) (*entity.ClusterConfig, error) {
	var (
		err    error
		filter *entity.ClusterConfig
	)

	ccBytes, err := cluster.Marshal()
	if err != nil {
		return nil, err
	}
	cluster.LastApply = string(ccBytes)

	// update
	if cluster.ID != 0 {
		filter = &entity.ClusterConfig{
			BaseModel: entity.BaseModel{
				ID: cluster.ID,
			},
		}
		original := &entity.ClusterConfig{}
		err = jms.clusterRepo.GetClustersInfo(filter, original)
		if err != nil {
			return nil, err
		}

		originalByte, err := original.Marshal()
		if err != nil {
			return nil, err
		}

		patch, err := jsonpatch.CreateMergePatch(originalByte, ccBytes)
		if err != nil {
			return nil, err
		}

		patcher, err := cluster.Unmarshal(patch)
		if err != nil {
			return nil, err
		}

		if !patcher.IsEmpty() {
			merge, err := jsonpatch.MergePatch(originalByte, patch)
			if err != nil {
				return nil, err
			}
			patcher.LastApply = string(merge)
			err = jms.clusterRepo.UpdateCluster(ctx, patcher)
			if err != nil {
				return nil, err
			}
			return cluster, nil
		}
	}
	// create
	err = jms.clusterRepo.CreateCluster(ctx, cluster)
	return cluster, err
}
