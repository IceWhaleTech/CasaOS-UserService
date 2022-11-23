package service

import (
	"github.com/IceWhaleTech/CasaOS-Common/external"
	"gorm.io/gorm"
)

var MyService Repository

type Repository interface {
	Gateway() external.ManagementService
	User() UserService
	Event() EventService
}

func NewService(db *gorm.DB, RuntimePath string) Repository {

	gatewayManagement, err := external.NewManagementService(RuntimePath)
	if err != nil {
		panic(err)
	}

	return &store{
		gateway: gatewayManagement,
		user:    NewUserService(db),
		event:   NewEventService(db),
	}
}

type store struct {
	gateway external.ManagementService
	user    UserService
	event   EventService
}

func (c *store) Event() EventService {
	return c.event
}
func (c *store) Gateway() external.ManagementService {
	return c.gateway
}

func (c *store) User() UserService {
	return c.user
}
