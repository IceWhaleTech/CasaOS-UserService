package config

import (
	"fmt"
	"log"
	"os"

	"github.com/IceWhaleTech/CasaOS-UserService/model"
	"gopkg.in/ini.v1"
)

// models with default values

var CommonInfo = &model.CommonModel{
	RuntimePath: "/var/run/casaos",
}

var AppInfo = &model.APPModel{
	DBPath:       "/var/lib/casaos",
	UserDataPath: "/var/lib/casaos",
	LogPath:      "/var/log/casaos",
	LogSaveName:  "user",
	LogFileExt:   "log",
}

var Cfg *ini.File

func InitSetup(config string) {
	configFilePath := UserServiceConfigFilePath
	if len(config) > 0 {
		configFilePath = config
	}

	var err error

	Cfg, err = ini.Load(configFilePath)
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}

	mapTo("common", CommonInfo)
	mapTo("app", AppInfo)
}

func SaveSetup(config string) {
	reflectFrom("common", CommonInfo)
	reflectFrom("app", AppInfo)

	configFilePath := UserServiceConfigFilePath
	if len(config) > 0 {
		configFilePath = config
	}

	if err := Cfg.SaveTo(configFilePath); err != nil {
		fmt.Printf("Fail to save file: %v", err)
		os.Exit(1)
	}
}

func mapTo(section string, v interface{}) {
	err := Cfg.Section(section).MapTo(v)
	if err != nil {
		log.Fatalf("Cfg.MapTo %s err: %v", section, err)
	}
}

func reflectFrom(section string, v interface{}) {
	err := Cfg.Section(section).ReflectFrom(v)
	if err != nil {
		log.Fatalf("Cfg.ReflectFrom %s err: %v", section, err)
	}
}
