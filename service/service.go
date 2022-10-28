package service

import (
	external "github.com/IceWhaleTech/CasaOS-Common/service/v1"
	"gorm.io/gorm"
)

var MyService Repository

type Repository interface {
	Gateway() external.ManagementService
	User() UserService
}

func NewService(db *gorm.DB, RuntimePath string) Repository {

	gatewayManagement, err := external.NewManagementService(RuntimePath)
	if err != nil {
		panic(err)
	}

	return &store{
		gateway: gatewayManagement,
		user:    NewUserService(db),
	}
}

type store struct {
	gateway external.ManagementService
	user    UserService
}

func (c *store) Gateway() external.ManagementService {
	return c.gateway
}

func (c *store) User() UserService {
	return c.user
}
