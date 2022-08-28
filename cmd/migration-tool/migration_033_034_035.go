package main

import (
	"os"

	interfaces "github.com/IceWhaleTech/CasaOS-Common"
	"github.com/IceWhaleTech/CasaOS-Common/utils/version"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/config"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/utils/file"
	"gopkg.in/ini.v1"
)

type migrationTool2 struct{}

func (u *migrationTool2) IsMigrationNeeded() (bool, error) {
	if _, err := os.Stat(version.LegacyCasaOSConfigFilePath); err != nil {
		_logger.Info("`%s` not found, migration is not needed.", version.LegacyCasaOSConfigFilePath)
		return false, nil
	}

	majorVersion, minorVersion, patchVersion, err := version.DetectLegacyVersion()
	if err != nil {
		if err == version.ErrLegacyVersionNotFound {
			return false, nil
		}

		return false, err
	}

	if majorVersion != 0 {
		return false, nil
	}

	if minorVersion != 3 {
		return false, nil
	}

	if patchVersion < 3 || patchVersion > 5 {
		return false, nil
	}

	// legacy version has to be between 0.3.3 and 0.3.5
	_logger.Info("Migration is needed for a CasaOS version between 0.3.3 and 0.3.5...")
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

	_logger.Info("Updating %s with settings from legacy configuration...", config.UserServiceConfigFilePath)
	config.InitSetup(config.UserServiceConfigFilePath)

	// LogPath
	if logPath, err := legacyConfigFile.Section("app").GetKey("LogPath"); err == nil {
		_logger.Info("[app] LogPath = %s", logPath.String())
		config.AppInfo.LogPath = logPath.Value()
	}

	// LogFileExt
	if logFileExt, err := legacyConfigFile.Section("app").GetKey("LogFileExt"); err == nil {
		_logger.Info("[app] LogFileExt = %s", logFileExt.String())
		config.AppInfo.LogFileExt = logFileExt.Value()
	}

	// DBPath
	if dbPath, err := legacyConfigFile.Section("app").GetKey("DBPath"); err == nil {
		_logger.Info("[app] DBPath = %s", dbPath.String())
		config.AppInfo.DBPath = dbPath.Value()
	}

	// UserDataPath
	if userDataPath, err := legacyConfigFile.Section("app").GetKey("UserDataPath"); err == nil {
		_logger.Info("[app] UserDataPath = %s", userDataPath.String())
		config.AppInfo.UserDataPath = userDataPath.Value()
	}

	_logger.Info("Saving %s...", config.UserServiceConfigFilePath)
	config.SaveSetup(config.UserServiceConfigFilePath)

	return nil
}

func (u *migrationTool2) PostMigrate() error {
	return nil
}

func NewMigrationToolFor033_034_035() interfaces.MigrationTool {
	return &migrationTool2{}
}
