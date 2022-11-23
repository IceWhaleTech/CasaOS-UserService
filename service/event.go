package service

import (
	"github.com/IceWhaleTech/CasaOS-UserService/model"
	"gorm.io/gorm"
)

type EventService interface {
	CreateEvemt(m model.EventModel) model.EventModel
	GetEvents() (list []model.EventModel)
	GetEventByUUID(uuid string) (m model.EventModel)
	DeleteEvent(uuid string)
}

type eventService struct {
	db *gorm.DB
}

func (e *eventService) CreateEvemt(m model.EventModel) model.EventModel {
	e.db.Create(&m)
	return m
}
func (e *eventService) GetEvents() (list []model.EventModel) {
	e.db.Find(&list)
	return
}
func (e *eventService) GetEventByUUID(uuid string) (m model.EventModel) {
	e.db.Where("uuid = ?", uuid).First(&m)
	return
}
func (e *eventService) DeleteEvent(uuid string) {
	e.db.Where("uuid = ?", uuid).Delete(&model.EventModel{})
}

func NewEventService(db *gorm.DB) EventService {
	return &eventService{db: db}
}
