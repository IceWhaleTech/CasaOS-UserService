//go:generate bash -c "mkdir -p codegen/user_service && go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.12.2 -generate types,server,spec -package codegen api/user_service/openapi.yaml > codegen/user_service/api.go"

package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/IceWhaleTech/CasaOS-Common/model"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-UserService/common"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/config"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/sqlite"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/utils/encryption"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/utils/random"
	"github.com/IceWhaleTech/CasaOS-UserService/route"
	"github.com/IceWhaleTech/CasaOS-UserService/service"
	"github.com/coreos/go-systemd/daemon"
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

	err = service.MyService.Gateway().CreateRoute(&model.Route{
		Path:   "/v1/users",
		Target: "http://" + listener.Addr().String(),
	})

	if err != nil {
		panic(err)
	}

	if supported, err := daemon.SdNotify(false, daemon.SdNotifyReady); err != nil {
		logger.Error("Failed to notify systemd that user service is ready", zap.Any("error", err))
	} else if supported {
		logger.Info("Notified systemd that user service is ready")
	} else {
		logger.Info("This process is not running as a systemd service.")
	}

	logger.Info("User service is listening...", zap.Any("address", listener.Addr().String()))

	s := &http.Server{
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second, // fix G112: Potential slowloris attack (see https://github.com/securego/gosec)
	}

	err = s.Serve(listener) // not using http.serve() to fix G114: Use of net/http serve function that has no support for setting timeouts (see https://github.com/securego/gosec)
	if err != nil {
		panic(err)
	}
}
