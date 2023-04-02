package service

import (
	"context"
	"github.com/daicheng123/kubejump/internal/entity"
)

type UserService struct {
	userRepo entity.UserRepo
}

func NewUserService(userRepo entity.UserRepo) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

func (us *UserService) GetUserInfoByName(ctx context.Context, username string) (*entity.User, error) {
	var user = &entity.User{}
	err := us.userRepo.GetInfoByName(ctx, username, user)

	if err != nil {
		return nil, err
	}

	return user, err
}
