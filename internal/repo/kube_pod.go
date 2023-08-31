package repo

import (
	"context"
	"errors"
	"github.com/daicheng123/kubejump/internal/base/data"
	"github.com/daicheng123/kubejump/internal/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sync"
)

type PodRepo struct {
	data *data.Data
	lock sync.Mutex
}

func (pr *PodRepo) CreateOrUpdatePod(_ context.Context, pod *entity.Pod) error {
	pr.lock.Lock()
	defer pr.lock.Unlock()

	conflictKeys := []clause.Column{
		{Name: "pod_name"},
		{Name: "namespace"},
		{Name: "cluster_uniq_key"},
	}
	tx := pr.data.DB.Session(&gorm.Session{}).Clauses(clause.OnConflict{
		UpdateAll: true,
		Columns:   conflictKeys,
	})
	return tx.Create(pod).Error
}

func (pr *PodRepo) ListPodsWithPreLoadCluster(_ context.Context, filter *entity.Pod, sortBy string) ([]*entity.Pod, error) {
	pr.lock.Lock()
	defer pr.lock.Unlock()

	if filter == nil {
		filter = &entity.Pod{}
	}

	var podList = make([]*entity.Pod, 0)
	db := pr.data.DB.Session(&gorm.Session{}).
		Model(filter).
		Preload("Cluster", IsActive(true)).
		//Scopes(IsActive(true)).
		Where(filter).
		Scopes(OrderBy(sortBy)).
		Find(&podList)

	return podList, db.Error
}

func (pr *PodRepo) PreloadPods(_ context.Context, filter *entity.Pod, reqParam *entity.PaginationParam) ([]*entity.Pod, error) {
	if filter == nil {
		filter = &entity.Pod{}
	}
	result := make([]*entity.Pod, 0)
	db := pr.data.DB.Session(&gorm.Session{}).
		Model(&entity.Pod{}).
		Preload("Cluster", IsActive(true)).
		Where(filter).
		Scopes(Paginate(reqParam.PageSize, reqParam.Offset), SearchBy(reqParam.Searches), OrderBy(reqParam.SortBy)).
		Find(&result)

	return result, db.Error
}

func (pr *PodRepo) CountPods(_ context.Context, filter *entity.Pod, reqParam *entity.PaginationParam) (int, error) {
	var count int64

	if filter == nil {
		filter = new(entity.Pod)
	}
	db := pr.data.DB.Session(&gorm.Session{}).Model(filter).Scopes(
		OrderBy(reqParam.SortBy),
		SearchBy(reqParam.Searches)).
		Count(&count)
	return int(count), db.Error
}

func (pr *PodRepo) DeletePodByNameAndNamespace(_ context.Context, name, ns string, id uint) error {
	if name == "" || id == 0 || ns == "" {
		return errors.New("name or uniqKey or namespace is Nil")
	}

	filter := &entity.Pod{
		PodName:    name,
		Namespace:  ns,
		ClusterRef: id,
	}

	pr.lock.Lock()
	defer pr.lock.Unlock()
	db := pr.data.DB.Session(&gorm.Session{}).Delete(filter)
	return db.Error
}

func NewPodRepo() entity.PodRepo {
	return &PodRepo{
		data: data.DefaultData,
	}
}
