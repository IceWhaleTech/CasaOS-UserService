package service

import (
	gateway "github.com/IceWhaleTech/CasaOS-Gateway/common"
	"gorm.io/gorm"
)

var MyService Repository

type Repository interface {
	Gateway() gateway.ManagementService
	User() UserService
}

func NewService(db *gorm.DB, RuntimePath string) Repository {

	gatewayManagement, err := gateway.NewManagementService(RuntimePath)
	if err != nil {
		panic(err)
	}

	return &store{
		gateway: gatewayManagement,
		user:    NewUserService(db),
	}
}

type store struct {
	gateway gateway.ManagementService
	user    UserService
}

func (c *store) Gateway() gateway.ManagementService {
	return c.gateway
}

func (c *store) User() UserService {
	return c.user
}
