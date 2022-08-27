package main

import (
	interfaces "github.com/IceWhaleTech/CasaOS-Common"
	"github.com/IceWhaleTech/CasaOS-Common/utils/version"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/config"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/utils/file"
	"gopkg.in/ini.v1"
)

type migrationTool2 struct{}

func (u *migrationTool2) IsMigrationNeeded() (bool, error) {
	_logger.Info("Checking if migration is needed for CasaoS version between 0.3.3 and 0.3.5...")

	minorVersion, err := version.DetectMinorVersion()
	if err != nil {
		return false, err
	}

	if minorVersion != 3 {
		return false, nil
	}

	// this is the best way to tell if CasaOS version is between 0.3.3 and 0.3.5
	isUserDataInDatabase, err := version.IsUserDataInDatabase()
	if err != nil {
		return false, err
	}

	if !isUserDataInDatabase {
		return false, nil
	}

	return true, nil
}

func (u *migrationTool2) PreMigrate() error {
	_logger.Info("Copying %s to %s if it doesn't exist...", userServiceConfigSampleFilePath, config.UserServiceConfigFilePath)
	if err := file.CopySingleFile(userServiceConfigSampleFilePath, config.UserServiceConfigFilePath, "skip"); err != nil {
		return err
	}
	return nil
}

func (u *migrationTool2) Migrate() error {
	_logger.Info("Loading legacy %s...", version.LegacyCasaOSConfigFilePath)
	legacyConfigFile, err := ini.Load(version.LegacyCasaOSConfigFilePath)
	if err != nil {
		return err
	}

	// LogPath
	logPath, err := legacyConfigFile.Section("app").GetKey("LogPath")
	if err != nil {
		return err
	}

	// LogFileExt
	logFileExt, err := legacyConfigFile.Section("app").GetKey("LogFileExt")
	if err != nil {
		return err
	}

	// DBPath
	dbPath, err := legacyConfigFile.Section("app").GetKey("DBPath")
	if err != nil {
		return err
	}

	// UserDataPath
	userDataPath, err := legacyConfigFile.Section("app").GetKey("UserDataPath")
	if err != nil {
		return err
	}

	_logger.Info("Updating %s with settings from legacy configuration...", config.UserServiceConfigFilePath)
	config.InitSetup(config.UserServiceConfigFilePath)

	config.AppInfo.LogPath = logPath.Value()
	config.AppInfo.LogFileExt = logFileExt.Value()
	config.AppInfo.DBPath = dbPath.Value()
	config.AppInfo.UserDataPath = userDataPath.Value()

	config.SaveSetup(config.UserServiceConfigFilePath)

	return nil
}

func (u *migrationTool2) PostMigrate() error {
	return nil
}

func NewMigrationToolFor033_034_035() interfaces.MigrationTool {
	return &migrationTool2{}
}
