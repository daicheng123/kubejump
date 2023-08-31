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

type NamespaceRepo struct {
	data *data.Data
	lock sync.Mutex
}

func (nsr *NamespaceRepo) CreateOrUpdateNS(_ context.Context, ns *entity.Namespace) error {
	nsr.lock.Lock()
	defer nsr.lock.Unlock()

	conflictKeys := []clause.Column{
		{Name: "namespace"},
		{Name: "cluster_uniq_key"},
	}
	tx := nsr.data.DB.Session(&gorm.Session{}).Clauses(clause.OnConflict{
		UpdateAll: true,
		Columns:   conflictKeys,
	})
	return tx.Create(ns).Error
}

func (nsr *NamespaceRepo) DeleteNSByName(_ context.Context, name, uniqKey string) error {
	if name == "" || uniqKey == "" {
		return errors.New("name or uniqKey is Nil")
	}

	filter := &entity.Namespace{
		NamespaceName:  name,
		ClusterUniqKey: uniqKey,
	}

	nsr.lock.Lock()
	defer nsr.lock.Unlock()
	db := nsr.data.DB.Session(&gorm.Session{}).Delete(filter)

	return db.Error
}

func NewNamespaceRepo() entity.NamespaceRepo {
	return &NamespaceRepo{
		//lock: sync.Mutex{},
		data: data.DefaultData,
	}
}
