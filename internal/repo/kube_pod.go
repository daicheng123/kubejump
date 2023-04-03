package repo

import (
	"context"
	"github.com/daicheng123/kubejump/internal/base/data"
	"github.com/daicheng123/kubejump/internal/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sync"
)

type PodRepo struct {
	data *data.Data
	lock sync.Locker
}

func (pr *PodRepo) AddPod(_ context.Context, pod *entity.Pod) error {
	return pr.data.DB.Session(&gorm.Session{}).Create(pod).Error
}

func (pr *PodRepo) UpdatePod(_ context.Context, pod *entity.Pod) error {
	return pr.data.DB.Session(&gorm.Session{}).Updates(pod).Error
}

func (pr *PodRepo) CreateOrUpdatePod(_ context.Context, pod *entity.Pod) error {
	pr.lock.Unlock()
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
