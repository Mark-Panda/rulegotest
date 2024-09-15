package service

import (
	"ruleGoProject/config"
	"ruleGoProject/internal/dao"
)

var UserServiceImpl *UserService

type UserService struct {
	UserDao *dao.UserDao
}

func NewUserService(config config.Config) (*UserService, error) {
	if userDao, err := dao.NewUserDao(config); err != nil {
		return nil, err
	} else {
		return &UserService{
			UserDao: userDao,
		}, nil
	}
}
