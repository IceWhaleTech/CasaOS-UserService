package model

type EventModel struct {
	UUID       string `gorm:"primaryKey" json:"uuid"`
	SourceID   string `gorm:"index" json:"source_id"`
	Name       string `json:"name"`
	Properties string `gorm:"serializer:json" json:"properties"`
	Timestamp  int64  `gorm:"autoCreateTime:milli" json:"timestamp"`
}

func (p *EventModel) TableName() string {
	return "events"
}
