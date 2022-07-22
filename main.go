package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/IceWhaleTech/CasaOS-UserService/pkg/config"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/utils/encryption"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/utils/logger"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/utils/random"
	"github.com/IceWhaleTech/CasaOS-UserService/route"
	"github.com/IceWhaleTech/CasaOS-UserService/service"
	"github.com/IceWhaleTech/CasaOS/pkg/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var sqliteDB *gorm.DB

var configFlag = flag.String("c", "", "config address")
var dbFlag = flag.String("db", "", "db path")
var resetUser = flag.Bool("ru", false, "reset user")
var user = flag.String("user", "", "user name")

func init() {
	flag.Parse()
	config.InitSetup(*configFlag)

	logger.LogInit()

	if len(*dbFlag) == 0 {
		*dbFlag = config.AppInfo.DBPath + "/db"
	}

	sqliteDB = sqlite.GetDb(*dbFlag)
	//gredis.GetRedisConn(config.RedisInfo),
	service.MyService = service.NewService(sqliteDB)
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

	s := &http.Server{
		Addr:           fmt.Sprintf(":%v", config.ServerInfo.HttpPort),
		Handler:        r,
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	logger.Info("UserService listening at port", zap.String("port", config.ServerInfo.HttpPort))

	s.ListenAndServe()
}
