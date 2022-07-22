package service

import "gorm.io/gorm"

var MyService Repository

type Repository interface {
	User() UserService
}

func NewService(db *gorm.DB) Repository {
	return &store{
		user: NewUserService(db),
	}
}

type store struct {
	user UserService
}

func (c *store) User() UserService {
	return c.user
}
