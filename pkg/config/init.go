package config

import (
	"fmt"
	"log"
	"os"

	"github.com/IceWhaleTech/CasaOS-UserService/model"
	"gopkg.in/ini.v1"
)

var ServerInfo = &model.ServerModel{}

//用户相关
var AppInfo = &model.APPModel{}

var Cfg *ini.File

func InitSetup(config string) {

	var configDir = USERCONFIGURL
	if len(config) > 0 {
		configDir = config
	}

	var err error
	//读取文件
	Cfg, err = ini.Load(configDir)
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}

	mapTo("app", AppInfo)
	mapTo("server", ServerInfo)
}

func mapTo(section string, v interface{}) {
	err := Cfg.Section(section).MapTo(v)
	if err != nil {
		log.Fatalf("Cfg.MapTo %s err: %v", section, err)
	}
}
