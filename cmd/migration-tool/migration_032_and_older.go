package main

import (
	"os"
	"path"
	"strconv"
	"strings"

	interfaces "github.com/IceWhaleTech/CasaOS-Common"
	"github.com/IceWhaleTech/CasaOS-Common/utils/version"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/config"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/sqlite"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/utils/encryption"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/utils/file"
	"github.com/IceWhaleTech/CasaOS-UserService/service"
	"github.com/IceWhaleTech/CasaOS-UserService/service/model"
	"gopkg.in/ini.v1"
)

type migrationTool1 struct{}

func (u *migrationTool1) IsMigrationNeeded() (bool, error) {
	_logger.Info("Checking if `%s` exists...", version.LegacyCasaOSConfigFilePath)
	if _, err := os.Stat(version.LegacyCasaOSConfigFilePath); err != nil {
		_logger.Info("`%s` not found, migration is not needed.", version.LegacyCasaOSConfigFilePath)
		return false, nil
	}

	_logger.Info("Checking if migration is needed for CasaoS version 0.3.2 or older...")

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

	if minorVersion == 2 {
		return true, nil
	}

	if minorVersion == 3 && patchVersion < 2 {
		return true, nil
	}

	return false, nil
}

func (u *migrationTool1) PreMigrate() error {
	_logger.Info("Copying %s to %s if it doesn't exist...", userServiceConfigSampleFilePath, config.UserServiceConfigFilePath)
	if err := file.CopySingleFile(userServiceConfigSampleFilePath, config.UserServiceConfigFilePath, "skip"); err != nil {
		return err
	}
	return nil
}

func (u *migrationTool1) Migrate() error {
	_logger.Info("Loading legacy %s...", version.LegacyCasaOSConfigFilePath)
	legacyConfigFile, err := ini.Load(version.LegacyCasaOSConfigFilePath)
	if err != nil {
		return err
	}

	_logger.Info("Updating %s with settings from legacy configuration...", config.UserServiceConfigFilePath)
	config.InitSetup(config.UserServiceConfigFilePath)

	// LogPath
	if logPath, err := legacyConfigFile.Section("app").GetKey("LogPath"); err == nil {
		_logger.Info("[app] LogPath = %s", logPath.Value())
		config.AppInfo.LogPath = logPath.Value()
	}

	if logPath, err := legacyConfigFile.Section("app").GetKey("LogSavePath"); err == nil {
		_logger.Info("[app] LogSavePath = %s", logPath.Value())
		config.AppInfo.LogPath = logPath.Value()
	}

	// LogFileExt
	if logFileExt, err := legacyConfigFile.Section("app").GetKey("LogFileExt"); err == nil {
		_logger.Info("[app] LogFileExt = %s", logFileExt.Value())
		config.AppInfo.LogFileExt = logFileExt.Value()
	}

	// UserDataPath
	if userDataPath, err := legacyConfigFile.Section("app").GetKey("UserDataPath"); err == nil {
		_logger.Info("[app] UserDataPath = %s", userDataPath.Value())
		config.AppInfo.UserDataPath = userDataPath.Value()
	}

	_logger.Info("Saving %s...", config.UserServiceConfigFilePath)
	config.SaveSetup(config.UserServiceConfigFilePath)

	_logger.Info("Migrating user from configuration file to database...")

	user := model.UserDBModel{}
	user.Role = "admin"

	// UserName
	if userName, err := legacyConfigFile.Section("user").GetKey("UserName"); err == nil {
		_logger.Info("[user] UserName = %s", userName.Value())
		user.Username = userName.Value()
	}

	// Email
	if userEmail, err := legacyConfigFile.Section("user").GetKey("Email"); err == nil {
		_logger.Info("[user] Email = %s", userEmail.Value())
		user.Email = userEmail.Value()
	}

	// NickName
	if userNickName, err := legacyConfigFile.Section("user").GetKey("NickName"); err == nil {
		_logger.Info("[user] NickName = %s", userNickName.Value())
		user.Nickname = userNickName.Value()
	}

	// Password
	if userPassword, err := legacyConfigFile.Section("user").GetKey("PWD"); err == nil {
		_logger.Info("[user] Password = %s", strings.Repeat("*", len(userPassword.Value())))
		user.Password = encryption.GetMD5ByStr(userPassword.Value())
	}

	sqliteDB := sqlite.GetDb(path.Join(config.AppInfo.DBPath, "user.db"))
	userService := service.NewUserService(sqliteDB)

	if len(user.Username) > 0 && userService.GetUserInfoByUserName(user.Username).Id == 0 {
		_logger.Info("Creating user %s in database...", user.Username)
		user = userService.CreateUser(user)
		if user.Id > 0 {
			userPath := config.AppInfo.UserDataPath + "/" + strconv.Itoa(user.Id)
			_logger.Info("Creating user data path: %s", userPath)
			file.MkDir(userPath)

			if legacyProjectPath, err := legacyConfigFile.Section("app").GetKey("ProjectPath"); err == nil {
				appOrderJsonFile := path.Join(legacyProjectPath.Value(), "app_order.json")

				if _, err := os.Stat(appOrderJsonFile); err == nil {
					_logger.Info("Moving %s to %s...", appOrderJsonFile, userPath)
					os.Rename(appOrderJsonFile, path.Join(userPath, "app_order.json"))
				}
			}
		}
	} else {
		_logger.Info("No user found, skipping...")
	}

	return nil
}

func (u *migrationTool1) PostMigrate() error {
	return nil
}

func NewMigrationToolFor032AndOlder() interfaces.MigrationTool {
	return &migrationTool1{}
}
