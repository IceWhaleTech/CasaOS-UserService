package route

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/IceWhaleTech/CasaOS-Common/external"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	message_bus "github.com/IceWhaleTech/CasaOS-UserService/codegen/message_bus"
	"github.com/IceWhaleTech/CasaOS-UserService/model"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/config"
	"github.com/IceWhaleTech/CasaOS-UserService/service"
	"go.uber.org/zap"
	"golang.org/x/net/websocket"
)

func EventListen() {

	messageBusUrl, err := external.GetMessageBusAddress(config.CommonInfo.RuntimePath)
	if err != nil {
		logger.Error("get message bus url error", zap.Any("err", err))
		return
	}

	wsURL := fmt.Sprintf("ws://%s/event/%s", strings.ReplaceAll(messageBusUrl, "http://", ""), "local-storage")
	ws, err := websocket.Dial(wsURL, "", "http://localhost")
	if err != nil {
		logger.Error("connect websocket err", zap.Any("error", err))
	}
	defer ws.Close()

	log.Println("subscribed to", wsURL)
	for {

		msg := make([]byte, 1024)
		n, err := ws.Read(msg)
		if err != nil {
			log.Fatalln(err.Error())
		}

		var event message_bus.Event

		if err := json.Unmarshal(msg[:n], &event); err != nil {
			log.Println(err.Error())
		}
		propertiesStr, err := json.Marshal(event.Properties)
		if err != nil {
			continue
		}
		model := model.EventModel{
			SourceID:   event.SourceID,
			Name:       event.Name,
			Properties: string(propertiesStr),
			UUID:       *event.Uuid,
		}
		service.MyService.Event().CreateEvemt(model)
		output, err := json.MarshalIndent(event, "", "  ")
		if err != nil {
			log.Println(err.Error())
		}
		log.Println(string(output))
	}
}
