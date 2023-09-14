package repo

import (
	"context"
	"errors"
	"github.com/daicheng123/kubejump/internal/base/data"
	"github.com/daicheng123/kubejump/internal/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strconv"
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
		{Name: "cluster_ref"},
	}
	tx := pr.data.DB.Session(&gorm.Session{}).Clauses(clause.OnConflict{
		UpdateAll: true,
		Columns:   conflictKeys,
	})
	return tx.Create(pod).Error
}
func validConvertNum(str string) (int64, error) {
	return strconv.ParseInt(str, 10, 64)

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
		Where(filter).
		Scopes(OrderBy(sortBy)).
		Find(&podList)
	return podList, db.Error
}

func (pr *PodRepo) PreloadPodsWithPager(_ context.Context, filter *entity.Pod, reqParam *entity.PaginationParam) ([]*entity.Pod, int, error) {
	var count int64
	if filter == nil {
		filter = &entity.Pod{}
	}
	result := make([]*entity.Pod, 0)
	db := pr.data.DB.Session(&gorm.Session{}).
		Model(&entity.Pod{}).
		Preload("Cluster", IsActive(reqParam.IsActive)).
		Where(filter).
		Scopes(
			OrderBy(reqParam.SortBy),
			SearchPodBy(reqParam.Search),
		)

	db.Count(&count)
	if db.Error != nil {
		return result, 0, db.Error
	}

	db = db.Scopes(PaginatePods(reqParam.PageSize, reqParam.Offset))
	db.Find(&result)

	return result, int(count), db.Error
}

func (pr *PodRepo) DeletePodByNameAndNamespace(_ context.Context, name, ns, cluster string) error {
	if name == "" || cluster == "" || ns == "" {
		return errors.New("name or uniqKey or namespace is Nil")
	}

	filter := &entity.Pod{
		PodName:    name,
		Namespace:  ns,
		ClusterRef: cluster,
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
