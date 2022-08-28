package main

import (
	"flag"
	"fmt"
	"os"

	interfaces "github.com/IceWhaleTech/CasaOS-Common"
	"github.com/IceWhaleTech/CasaOS-Common/utils/systemctl"
	"github.com/IceWhaleTech/CasaOS-UserService/common"
)

const (
	userServiceConfigSampleFilePath = "/etc/casaos/user-service.conf.sample"
	userServiceName                 = "casaos-user-service.service"
)

var _logger *Logger

func main() {
	versionFlag := flag.Bool("v", false, "version")
	debugFlag := flag.Bool("d", true, "debug")
	forceFlag := flag.Bool("f", false, "force")
	flag.Parse()

	if *versionFlag {
		fmt.Println(common.Version)
		os.Exit(0)
	}

	_logger = NewLogger()

	if os.Getuid() != 0 {
		_logger.Info("Root privileges are required to run this program.")
		os.Exit(1)
	}

	if *debugFlag {
		_logger.DebugMode = true
	}

	if !*forceFlag {
		isRunning, err := systemctl.IsServiceRunning(userServiceName)
		if err != nil {
			_logger.Error("Failed to check if %s is enabled", userServiceName)
			panic(err)
		}

		if isRunning {
			_logger.Info("%s is running. If migration is still needed, try with -f.", userServiceName)
			os.Exit(1)
		}
	}

	migrationTools := []interfaces.MigrationTool{
		NewMigrationToolFor032AndOlder(),
		NewMigrationToolFor033_034_035(),
	}

	var selectedMigrationTool interfaces.MigrationTool

	// look for the right migration tool matching current version
	for _, tool := range migrationTools {
		migrationNeeded, err := tool.IsMigrationNeeded()
		if err != nil {
			panic(err)
		}

		if migrationNeeded {
			selectedMigrationTool = tool
			break
		}
	}

	if selectedMigrationTool == nil {
		_logger.Info("No migration to proceed.")
		return
	}

	if err := selectedMigrationTool.PreMigrate(); err != nil {
		panic(err)
	}

	if err := selectedMigrationTool.Migrate(); err != nil {
		panic(err)
	}

	if err := selectedMigrationTool.PostMigrate(); err != nil {
		panic(err)
	}
}
