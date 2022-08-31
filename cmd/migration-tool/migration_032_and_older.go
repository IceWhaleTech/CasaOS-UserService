package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

	if minorVersion == 2 {
		_logger.Info("Migration is needed for a CasaOS version 0.2.x...")
		return true, nil
	}

	if minorVersion == 3 && patchVersion < 2 {
		_logger.Info("Migration is needed for a CasaOS version between 0.3.0 and 0.3.2...")
		return true, nil
	}

	return false, nil
}

func (u *migrationTool1) PreMigrate() error {
	if _, err := os.Stat(userServiceConfigDirPath); os.IsNotExist(err) {
		_logger.Info("Creating %s since it doesn't exists...", userServiceConfigDirPath)
		if err := os.Mkdir(userServiceConfigDirPath, 0o755); err != nil {
			return err
		}
	}

	if _, err := os.Stat(userServiceConfigFilePath); os.IsNotExist(err) {
		_logger.Info("Creating %s since it doesn't exist...", userServiceConfigFilePath)

		f, err := os.Create(userServiceConfigFilePath)
		if err != nil {
			return err
		}

		defer f.Close()

		if _, err := f.WriteString(_userServiceConfigFileSample); err != nil {
			return err
		}
	}

	extension := "." + time.Now().Format("20060102") + ".bak"

	_logger.Info("Creating a backup %s if it doesn't exist...", version.LegacyCasaOSConfigFilePath+extension)
	return file.CopySingleFile(version.LegacyCasaOSConfigFilePath, version.LegacyCasaOSConfigFilePath+extension, "skip")
}

func (u *migrationTool1) Migrate() error {
	_logger.Info("Loading legacy %s...", version.LegacyCasaOSConfigFilePath)
	legacyConfigFile, err := ini.Load(version.LegacyCasaOSConfigFilePath)
	if err != nil {
		return err
	}

	migrateConfigurationFile1(legacyConfigFile)

	return migrateUser1(legacyConfigFile)
}

func (u *migrationTool1) PostMigrate() error {
	legacyConfigFile, err := ini.Load(version.LegacyCasaOSConfigFilePath)
	if err != nil {
		return err
	}

	_logger.Info("Deleting legacy `user` section in %s...", version.LegacyCasaOSConfigFilePath)

	legacyConfigFile.DeleteSection("user")

	if err := legacyConfigFile.SaveTo(version.LegacyCasaOSConfigFilePath); err != nil {
		return err
	}

	dbPath := legacyConfigFile.Section("app").Key("DBPath").String()

	dbFile := filepath.Join(dbPath, "db", "casaOS.db")

	if _, err := os.Stat(dbFile); err != nil {
		dbFile = filepath.Join(defaultDBPath, "db", "casaOS.db")

		if _, err := os.Stat(dbFile); err != nil {
			return nil
		}
	}

	legacyDB, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return err
	}

	defer legacyDB.Close()

	for _, tableName := range []string{"o_users", "o_user"} {
		tableExists, err := isTableExist(legacyDB, tableName)
		if err != nil {
			return err
		}

		if !tableExists {
			continue
		}

		_logger.Info("Dropping `%s` table in legacy database...", tableName)

		if _, err = legacyDB.Exec("DROP TABLE " + tableName); err != nil {
			_logger.Error("Failed to drop `%s` table in legacy database: %s", tableName, err)
		}
	}

	return nil
}

func NewMigrationToolFor032AndOlder() interfaces.MigrationTool {
	return &migrationTool1{}
}

func migrateConfigurationFile1(legacyConfigFile *ini.File) {
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
}

func migrateUser1(legacyConfigFile *ini.File) error {
	_logger.Info("Migrating user from configuration file to database...")

	user := model.UserDBModel{Role: "admin"}

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

	newDB := sqlite.GetDb(config.AppInfo.DBPath)
	userService := service.NewUserService(newDB)

	if len(user.Username) == 0 {
		_logger.Info("No user found in legacy configuration file. Skipping...")
		return nil
	}

	if userService.GetUserInfoByUserName(user.Username).Id > 0 {
		_logger.Info("User `%s` already exists in user database at %s. Skipping...", user.Username, config.AppInfo.DBPath)
		return nil
	}

	_logger.Info("Creating user %s in database at %s...", user.Username, config.AppInfo.DBPath)
	user = userService.CreateUser(user)
	if user.Id > 0 {
		userPath := config.AppInfo.UserDataPath + "/" + strconv.Itoa(user.Id)
		_logger.Info("Creating user data path: %s", userPath)
		if err := file.MkDir(userPath); err != nil {
			return err
		}

		if legacyProjectPath, err := legacyConfigFile.Section("app").GetKey("ProjectPath"); err == nil {
			appOrderJSONFile := filepath.Join(legacyProjectPath.Value(), "app_order.json")

			if _, err := os.Stat(appOrderJSONFile); err == nil {
				_logger.Info("Moving %s to %s...", appOrderJSONFile, userPath)
				if err := os.Rename(appOrderJSONFile, filepath.Join(userPath, "app_order.json")); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
