package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	interfaces "github.com/IceWhaleTech/CasaOS-Common"
	"github.com/IceWhaleTech/CasaOS-Common/utils/version"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/config"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/sqlite"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/utils/file"
	"github.com/IceWhaleTech/CasaOS-UserService/service"
	"github.com/IceWhaleTech/CasaOS-UserService/service/model"
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

	extension := "." + time.Now().Format("20060102") + ".bak"

	_logger.Info("Creating a backup %s if it doesn't exist...", version.LegacyCasaOSConfigFilePath+extension)
	if err := file.CopySingleFile(version.LegacyCasaOSConfigFilePath, version.LegacyCasaOSConfigFilePath+extension, "skip"); err != nil {
		return err
	}

	legacyConfigFile, err := ini.Load(version.LegacyCasaOSConfigFilePath)
	if err != nil {
		return err
	}

	dbPath := legacyConfigFile.Section("app").Key("DBPath").String()

	dbFile := filepath.Join(dbPath, "db", "casaOS.db")

	_logger.Info("Creating a backup %s if it doesn't exist...", dbFile+extension)
	if err := file.CopySingleFile(dbFile, dbFile+extension, "skip"); err != nil {
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

	migrateConfigurationFile2(legacyConfigFile)

	return migrateUser2(legacyConfigFile)
}

func (u *migrationTool2) PostMigrate() error {
	legacyConfigFile, err := ini.Load(version.LegacyCasaOSConfigFilePath)
	if err != nil {
		return err
	}

	dbPath := legacyConfigFile.Section("app").Key("DBPath").String()

	dbFile := filepath.Join(dbPath, "db", "casaOS.db")

	if _, err := os.Stat(dbFile); err != nil {
		return err
	}

	legacyDB, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return err
	}

	defer legacyDB.Close()

	if tableExists, err := isTableExist(legacyDB, "o_users"); err != nil {
		return err
	} else if tableExists {
		_logger.Info("Dropping `o_users` table in legacy database...")

		if _, err = legacyDB.Exec("DROP TABLE o_users"); err != nil {
			_logger.Error("Failed to drop `o_users` table in legacy database: %s", err)
		}
	}
	return nil
}

func NewMigrationToolFor033_034_035() interfaces.MigrationTool {
	return &migrationTool2{}
}

func migrateConfigurationFile2(legacyConfigFile *ini.File) {
	_logger.Info("Updating %s with settings from legacy configuration...", config.UserServiceConfigFilePath)
	config.InitSetup(config.UserServiceConfigFilePath)

	// LogPath
	if logPath, err := legacyConfigFile.Section("app").GetKey("LogPath"); err == nil {
		_logger.Info("[app] LogPath = %s", logPath.Value())
		config.AppInfo.LogPath = logPath.Value()
	}

	// LogFileExt
	if logFileExt, err := legacyConfigFile.Section("app").GetKey("LogFileExt"); err == nil {
		_logger.Info("[app] LogFileExt = %s", logFileExt.Value())
		config.AppInfo.LogFileExt = logFileExt.Value()
	}

	// DBPath
	if dbPath, err := legacyConfigFile.Section("app").GetKey("DBPath"); err == nil {
		_logger.Info("[app] DBPath = %s", dbPath.Value())
		config.AppInfo.DBPath = dbPath.Value() + "/db"
	}

	// UserDataPath
	if userDataPath, err := legacyConfigFile.Section("app").GetKey("UserDataPath"); err == nil {
		_logger.Info("[app] UserDataPath = %s", userDataPath.Value())
		config.AppInfo.UserDataPath = userDataPath.Value()
	}

	_logger.Info("Saving %s...", config.UserServiceConfigFilePath)
	config.SaveSetup(config.UserServiceConfigFilePath)
}

func migrateUser2(legacyConfigFile *ini.File) error {
	_logger.Info("Migrating user from legacy database to user database...")

	user := model.UserDBModel{Role: "admin"}

	dbPath := legacyConfigFile.Section("app").Key("DBPath").String()

	dbFile := filepath.Join(dbPath, "db", "casaOS.db")

	if _, err := os.Stat(dbFile); err != nil {
		return err
	}

	legacyDB, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return err
	}

	defer legacyDB.Close()

	if tableExists, err := isTableExist(legacyDB, "o_users"); err != nil {
		return err
	} else if !tableExists {
		_logger.Info("Table `o_users` not found in legacy database. Skipping...")
		return nil
	}

	sqlStatement := "SELECT id, username, password, role, email, nickname, avatar, description, created_at FROM o_users ORDER BY id ASC"

	rows, err := legacyDB.Query(sqlStatement)
	if err != nil {
		return err
	}

	defer rows.Close()

	newDB := sqlite.GetDb(config.AppInfo.DBPath)
	userService := service.NewUserService(newDB)

	for rows.Next() {
		if err := rows.Scan(
			&user.Id,
			&user.Username,
			&user.Password,
			&user.Role,
			&user.Email,
			&user.Nickname,
			&user.Avatar,
			&user.Description,
			&user.CreatedAt,
		); err != nil {
			return err
		}

		if userService.GetUserAllInfoByName(user.Username).Id > 0 {
			_logger.Info("User %s already exists in user database at %s, skipping...", user.Username, config.AppInfo.DBPath)
			continue
		}

		_logger.Info("Creating user %s in user database...", user.Username)
		user = userService.CreateUser(user)
	}

	return nil
}

func isTableExist(legacyDB *sql.DB, tableName string) (bool, error) {
	rows, err := legacyDB.Query("SELECT name FROM sqlite_master WHERE type='table' AND name= ?", tableName)
	if err != nil {
		return false, err
	}

	defer rows.Close()

	return rows.Next(), nil
}
