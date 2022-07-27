package config

import (
	"fmt"
	"log"
	"os"

	"github.com/IceWhaleTech/CasaOS-UserService/model"
	"gopkg.in/ini.v1"
)

var CommonInfo = &model.CommonModel{}

var AppInfo = &model.APPModel{}

var Cfg *ini.File

func InitSetup(config string) {

	var configDir = USERCONFIGURL
	if len(config) > 0 {
		configDir = config
	}

	var err error

	Cfg, err = ini.Load(configDir)
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}

	mapTo("common", CommonInfo)
	mapTo("app", AppInfo)
}

func mapTo(section string, v interface{}) {
	err := Cfg.Section(section).MapTo(v)
	if err != nil {
		log.Fatalf("Cfg.MapTo %s err: %v", section, err)
	}
}
