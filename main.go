package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-Gateway/common"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/config"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/sqlite"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/utils/encryption"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/utils/random"
	"github.com/IceWhaleTech/CasaOS-UserService/route"
	"github.com/IceWhaleTech/CasaOS-UserService/service"
	"gorm.io/gorm"
)

var sqliteDB *gorm.DB

var (
	configFlag = flag.String("c", "", "config address")
	dbFlag     = flag.String("db", "", "db path")
	resetUser  = flag.Bool("ru", false, "reset user")
	user       = flag.String("user", "", "user name")
)

func init() {
	flag.Parse()
	config.InitSetup(*configFlag)

	logger.LogInit(config.AppInfo.LogPath, config.AppInfo.LogSaveName, config.AppInfo.LogFileExt)

	if len(*dbFlag) == 0 {
		*dbFlag = config.AppInfo.DBPath + "/db"
	}

	sqliteDB = sqlite.GetDb(*dbFlag)
	service.MyService = service.NewService(sqliteDB, config.CommonInfo.RuntimePath)
}

func main() {
	r := route.InitRouter()

	if *resetUser {
		if user == nil || len(*user) == 0 {
			fmt.Println("user is empty")
			return
		}
		userData := service.MyService.User().GetUserAllInfoByName(*user)
		if userData.Id == 0 {
			fmt.Println("user not exist")
			return
		}
		password := random.RandomString(6, false)
		userData.Password = encryption.GetMD5ByStr(password)
		service.MyService.User().UpdateUserPassword(userData)
		fmt.Println("User reset successful")
		fmt.Println("UserName:" + userData.Username)
		fmt.Println("Password:" + password)
		return
	}

	listener, err := net.Listen("tcp", "127.1:0")
	if err != nil {
		panic(err)
	}

	err = service.MyService.Gateway().CreateRoute(&common.Route{
		Path:   "/v1/users",
		Target: "http://" + listener.Addr().String(),
	})

	if err != nil {
		panic(err)
	}

	log.Printf("user service listening on %s", listener.Addr().String())
	err = http.Serve(listener, r)
	if err != nil {
		panic(err)
	}
}
