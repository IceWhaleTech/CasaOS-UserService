package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-Gateway/common"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/config"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/sqlite"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/utils/encryption"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/utils/random"
	"github.com/IceWhaleTech/CasaOS-UserService/route"
	"github.com/IceWhaleTech/CasaOS-UserService/service"
	"go.uber.org/zap"
)

const localhost = "127.0.0.1"

func init() {
	configFlag := flag.String("c", "", "config address")
	dbFlag := flag.String("db", "", "db path")
	resetUserFlag := flag.Bool("ru", false, "reset user")
	userFlag := flag.String("user", "", "user name")
	versionFlag := flag.Bool("v", false, "version")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("v%s\n", common.Version)
		os.Exit(0)
	}

	config.InitSetup(*configFlag)

	logger.LogInit(config.AppInfo.LogPath, config.AppInfo.LogSaveName, config.AppInfo.LogFileExt)

	if len(*dbFlag) == 0 {
		*dbFlag = config.AppInfo.DBPath
	}

	sqliteDB := sqlite.GetDb(*dbFlag)
	service.MyService = service.NewService(sqliteDB, config.CommonInfo.RuntimePath)

	if *resetUserFlag {
		if userFlag == nil || len(*userFlag) == 0 {
			fmt.Println("user is empty")
			return
		}

		userData := service.MyService.User().GetUserAllInfoByName(*userFlag)

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
	}
}

func main() {
	r := route.InitRouter()

	listener, err := net.Listen("tcp", net.JoinHostPort(localhost, "0"))
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

	logger.Info("User service is listening...", zap.Any("address", listener.Addr().String()))
	err = http.Serve(listener, r)
	if err != nil {
		panic(err)
	}
}
