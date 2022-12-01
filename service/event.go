package service

import (
	"encoding/json"

	"github.com/IceWhaleTech/CasaOS-UserService/model"
	"gorm.io/gorm"
)

type EventService interface {
	CreateEvemt(m model.EventModel) model.EventModel
	GetEvents() (list []model.EventModel)
	GetEventByUUID(uuid string) (m model.EventModel)
	DeleteEvent(uuid string)
	DeleteEventBySerial(serial string)
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
func (e *eventService) DeleteEventBySerial(serial string) {
	list := []model.EventModel{}
	e.db.Find(&list)
	for _, v := range list {

		if v.SourceID == "local-storage" {
			properties := make(map[string]string)
			err := json.Unmarshal([]byte(v.Properties), &properties)
			if err != nil {
				continue
			}
			if properties["serial"] == serial {
				e.db.Delete(&v)
			}
		}
	}
}
func NewEventService(db *gorm.DB) EventService {
	return &eventService{db: db}
}
