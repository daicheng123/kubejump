package repo

import (
	"context"
	"errors"
	"github.com/daicheng123/kubejump/internal/base/data"
	"github.com/daicheng123/kubejump/internal/entity"
	"gorm.io/gorm"
)

type UserRepo struct {
	data *data.Data
}

func (ur *UserRepo) GetInfoByID(_ context.Context, filter *entity.User) (*entity.User, error) {
	if filter == nil {
		return nil, errors.New("filter is Nil")
	}

	var userInfo *entity.User
	db := ur.data.DB.Session(&gorm.Session{}).Where(filter).Take(userInfo)
	return userInfo, db.Error
}

func (ur *UserRepo) GetInfoByName(_ context.Context, username string, user *entity.User) error {
	if user == nil {
		user = new(entity.User)
	}
	return ur.data.DB.Session(&gorm.Session{}).Where("username=?", username).Take(user).Error
}

func NewUserRepo() entity.UserRepo {
	return &UserRepo{
		data: data.DefaultData,
	}
}
