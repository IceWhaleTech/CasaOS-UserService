package v1

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	json2 "encoding/json"
	"image"
	"image/png"
	"io"
	"log"
	"net/http"
	url2 "net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/IceWhaleTech/CasaOS-Common/external"
	"github.com/IceWhaleTech/CasaOS-Common/utils/common_err"
	"github.com/IceWhaleTech/CasaOS-Common/utils/jwt"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-UserService/common"
	"github.com/IceWhaleTech/CasaOS-UserService/model"
	"github.com/IceWhaleTech/CasaOS-UserService/model/system_model"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/config"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/utils/encryption"
	"github.com/IceWhaleTech/CasaOS-UserService/pkg/utils/file"
	model2 "github.com/IceWhaleTech/CasaOS-UserService/service/model"
	"github.com/labstack/echo/v4"
	uuid "github.com/satori/go.uuid"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/IceWhaleTech/CasaOS-UserService/service"
)

// @Summary register user
// @Router /user/register/ [post]
func PostUserRegister(ctx echo.Context) error {
	json := make(map[string]string)
	ctx.Bind(&json)

	username := json["username"]
	pwd := json["password"]
	key := json["key"]
	if _, ok := service.UserRegisterHash[key]; !ok {
		return ctx.JSON(common_err.CLIENT_ERROR,
			model.Result{Success: common_err.KEY_NOT_EXIST, Message: common_err.GetMsg(common_err.KEY_NOT_EXIST)})
	}

	if len(username) == 0 || len(pwd) == 0 {
		return ctx.JSON(common_err.CLIENT_ERROR,
			model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
	}
	if len(pwd) < 6 {
		return ctx.JSON(common_err.CLIENT_ERROR,
			model.Result{Success: common_err.PWD_IS_TOO_SIMPLE, Message: common_err.GetMsg(common_err.PWD_IS_TOO_SIMPLE)})
	}
	oldUser := service.MyService.User().GetUserInfoByUserName(username)
	if oldUser.Id > 0 {
		return ctx.JSON(common_err.CLIENT_ERROR,
			model.Result{Success: common_err.USER_EXIST, Message: common_err.GetMsg(common_err.USER_EXIST)})
	}

	user := model2.UserDBModel{}
	user.Username = username
	user.Password = encryption.GetMD5ByStr(pwd)
	user.Role = "admin"

	user = service.MyService.User().CreateUser(user)
	if user.Id == 0 {
		return ctx.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.SERVICE_ERROR, Message: common_err.GetMsg(common_err.SERVICE_ERROR)})
	}
	file.MkDir(config.AppInfo.UserDataPath + "/" + strconv.Itoa(user.Id))
	delete(service.UserRegisterHash, key)
	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS)})
}

var limiter = rate.NewLimiter(rate.Every(time.Minute), 5)

// @Summary login
// @Produce  application/json
// @Accept application/json
// @Tags user
// @Param user_name query string true "User name"
// @Param pwd  query string true "password"
// @Success 200 {string} string "ok"
// @Router /user/login [post]
func PostUserLogin(ctx echo.Context) error {
	if !limiter.Allow() {
		return ctx.JSON(common_err.TOO_MANY_REQUEST,
			model.Result{
				Success: common_err.TOO_MANY_LOGIN_REQUESTS,
				Message: common_err.GetMsg(common_err.TOO_MANY_LOGIN_REQUESTS),
			})
	}

	json := make(map[string]string)
	ctx.Bind(&json)

	username := json["username"]

	password := json["password"]
	// check params is empty
	if len(username) == 0 || len(password) == 0 {
		return ctx.JSON(common_err.CLIENT_ERROR,
			model.Result{
				Success: common_err.CLIENT_ERROR,
				Message: common_err.GetMsg(common_err.INVALID_PARAMS),
			})
	}
	user := service.MyService.User().GetUserAllInfoByName(username)
	if user.Id == 0 {
		return ctx.JSON(common_err.CLIENT_ERROR,
			model.Result{Success: common_err.USER_NOT_EXIST, Message: common_err.GetMsg(common_err.USER_NOT_EXIST)})
	}
	if user.Password != encryption.GetMD5ByStr(password) {
		return ctx.JSON(common_err.CLIENT_ERROR,
			model.Result{Success: common_err.USER_NOT_EXIST_OR_PWD_INVALID, Message: common_err.GetMsg(common_err.USER_NOT_EXIST_OR_PWD_INVALID)})
	}

	privateKey, _ := service.MyService.User().GetKeyPair()

	token := system_model.VerifyInformation{}

	accessToken, err := jwt.GetAccessToken(user.Username, privateKey, user.Id)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, model.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
	}
	token.AccessToken = accessToken

	refreshToken, err := jwt.GetRefreshToken(user.Username, privateKey, user.Id)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, model.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
	}
	token.RefreshToken = refreshToken

	token.ExpiresAt = time.Now().Add(3 * time.Hour * time.Duration(1)).Unix()
	data := make(map[string]interface{}, 2)
	user.Password = ""
	data["token"] = token

	// TODO:1 Database fields cannot be external
	data["user"] = user

	return ctx.JSON(common_err.SUCCESS,
		model.Result{
			Success: common_err.SUCCESS,
			Message: common_err.GetMsg(common_err.SUCCESS),
			Data:    data,
		})
}

// @Summary edit user head
// @Produce  application/json
// @Accept multipart/form-data
// @Tags user
// @Param file formData file true "用户头像"
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /users/avatar [put]
func PutUserAvatar(ctx echo.Context) error {
	id := ctx.Request().Header.Get("user_id")
	user := service.MyService.User().GetUserInfoById(id)
	if user.Id == 0 {
		return ctx.JSON(common_err.SERVICE_ERROR,
			model.Result{Success: common_err.USER_NOT_EXIST, Message: common_err.GetMsg(common_err.USER_NOT_EXIST)})
	}
	json := make(map[string]string)
	ctx.Bind(&json)

	data := json["file"]
	imgBase64 := strings.Replace(data, "data:image/png;base64,", "", 1)
	decodeData, err := base64.StdEncoding.DecodeString(string(imgBase64))
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, model.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
	}

	// 将字节数组转为图片
	img, _, err := image.Decode(strings.NewReader(string(decodeData)))
	if err != nil {
		log.Fatal(err)
	}

	ext := ".png"
	avatarPath := config.AppInfo.UserDataPath + "/" + id + "/avatar" + ext
	os.Remove(avatarPath)
	outFile, err := os.Create(avatarPath)
	if err != nil {
		logger.Error("create file error", zap.Error(err))
	}
	defer outFile.Close()

	err = png.Encode(outFile, img)
	if err != nil {
		logger.Error("encode error", zap.Error(err))
	}
	user.Avatar = avatarPath
	service.MyService.User().UpdateUser(user)
	return ctx.JSON(http.StatusOK,
		model.Result{
			Success: common_err.SUCCESS,
			Message: common_err.GetMsg(common_err.SUCCESS),
			Data:    user,
		})
}

// @Summary get user head
// @Produce  application/json
// @Tags user
// @Param file formData file true "用户头像"
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /users/avatar [get]
func GetUserAvatar(ctx echo.Context) error {
	id := ctx.Request().Header.Get("user_id")
	user := service.MyService.User().GetUserInfoById(id)
	if user.Id == 0 {
		return ctx.JSON(common_err.SERVICE_ERROR,
			model.Result{Success: common_err.USER_NOT_EXIST, Message: common_err.GetMsg(common_err.USER_NOT_EXIST)})
	}

	if file.Exists(user.Avatar) {
		ctx.Response().Header().Set("Content-Disposition", "attachment; filename*=utf-8''"+url2.PathEscape(path.Base(user.Avatar)))
		ctx.Response().Header().Set("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate, value")
		return ctx.File(user.Avatar)

	}
	user.Avatar = "/usr/share/casaos/www/avatar.svg"
	if file.Exists(user.Avatar) {
		ctx.Response().Header().Set("Content-Disposition", "attachment; filename*=utf-8''"+url2.PathEscape(path.Base(user.Avatar)))
		ctx.Response().Header().Set("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate, value")
		return ctx.File(user.Avatar)

	}
	user.Avatar = "/var/lib/casaos/www/avatar.svg"
	ctx.Response().Header().Set("Content-Disposition", "attachment; filename*=utf-8''"+url2.PathEscape(path.Base(user.Avatar)))
	ctx.Response().Header().Set("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate, value")
	return ctx.File(user.Avatar)
}

// @Summary edit user name
// @Produce  application/json
// @Accept application/json
// @Tags user
// @Param old_name  query string true "Old user name"
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /user/name/:id [put]
func PutUserInfo(ctx echo.Context) error {
	id := ctx.Request().Header.Get("user_id")
	json := model2.UserDBModel{}
	ctx.Bind(&json)
	user := service.MyService.User().GetUserInfoById(id)
	if user.Id == 0 {
		return ctx.JSON(common_err.SERVICE_ERROR,
			model.Result{Success: common_err.USER_NOT_EXIST_OR_PWD_INVALID, Message: common_err.GetMsg(common_err.USER_NOT_EXIST_OR_PWD_INVALID)})
	}
	if len(json.Username) > 0 {
		u := service.MyService.User().GetUserInfoByUserName(json.Username)
		if u.Id > 0 {
			return ctx.JSON(common_err.CLIENT_ERROR,
				model.Result{Success: common_err.USER_EXIST, Message: common_err.GetMsg(common_err.USER_EXIST)})
		}
	}

	if len(json.Email) == 0 {
		json.Email = user.Email
	}
	if len(json.Avatar) == 0 {
		json.Avatar = user.Avatar
	}
	if len(json.Role) == 0 {
		json.Role = user.Role
	}
	if len(json.Description) == 0 {
		json.Description = user.Description
	}
	if len(json.Nickname) == 0 {
		json.Nickname = user.Nickname
	}
	service.MyService.User().UpdateUser(json)
	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: json})
}

// @Summary edit user password
// @Produce  application/json
// @Accept application/json
// @Tags user
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /user/password/:id [put]
func PutUserPassword(ctx echo.Context) error {
	id := ctx.Request().Header.Get("user_id")
	json := make(map[string]string)
	ctx.Bind(&json)
	oldPwd := json["old_password"]
	pwd := json["password"]
	if len(oldPwd) == 0 || len(pwd) == 0 {
		return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
	}
	user := service.MyService.User().GetUserAllInfoById(id)
	if user.Id == 0 {
		return ctx.JSON(common_err.SERVICE_ERROR,
			model.Result{Success: common_err.USER_NOT_EXIST, Message: common_err.GetMsg(common_err.USER_NOT_EXIST)})
	}
	if user.Password != encryption.GetMD5ByStr(oldPwd) {
		return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.PWD_INVALID_OLD, Message: common_err.GetMsg(common_err.PWD_INVALID_OLD)})
	}
	user.Password = encryption.GetMD5ByStr(pwd)
	service.MyService.User().UpdateUserPassword(user)
	user.Password = ""
	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: user})
}

// @Summary edit user nick
// @Produce  application/json
// @Accept application/json
// @Tags user
// @Param nick_name query string false "nick name"
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /user/nick [put]
func PutUserNick(ctx echo.Context) error {
	id := ctx.Request().Header.Get("user_id")
	json := make(map[string]string)
	ctx.Bind(&json)
	Nickname := json["nick_name"]
	if len(Nickname) == 0 {
		return ctx.JSON(http.StatusOK, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
	}
	user := service.MyService.User().GetUserInfoById(id)
	if user.Id == 0 {
		return ctx.JSON(http.StatusOK,
			model.Result{Success: common_err.USER_NOT_EXIST, Message: common_err.GetMsg(common_err.USER_NOT_EXIST)})
	}
	user.Nickname = Nickname
	service.MyService.User().UpdateUser(user)
	return ctx.JSON(http.StatusOK, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: user})
}

// @Summary edit user description
// @Produce  application/json
// @Accept multipart/form-data
// @Tags user
// @Param description formData string false "Description"
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /user/desc [put]
func PutUserDesc(ctx echo.Context) error {
	id := ctx.Request().Header.Get("user_id")
	json := make(map[string]string)
	ctx.Bind(&json)
	desc := json["description"]
	if len(desc) == 0 {
		return ctx.JSON(http.StatusOK, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
	}
	user := service.MyService.User().GetUserInfoById(id)
	if user.Id == 0 {
		return ctx.JSON(http.StatusOK,
			model.Result{Success: common_err.USER_NOT_EXIST, Message: common_err.GetMsg(common_err.USER_NOT_EXIST)})
	}
	user.Description = desc

	service.MyService.User().UpdateUser(user)

	return ctx.JSON(http.StatusOK, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: user})
}

// @Summary get user info
// @Produce  application/json
// @Accept  application/json
// @Tags user
// @Success 200 {string} string "ok"
// @Router /user/info/:id [get]
func GetUserInfo(ctx echo.Context) error {
	id := ctx.Request().Header.Get("user_id")
	user := service.MyService.User().GetUserInfoById(id)

	return ctx.JSON(common_err.SUCCESS,
		model.Result{
			Success: common_err.SUCCESS,
			Message: common_err.GetMsg(common_err.SUCCESS),
			Data:    user,
		})
}

/**
 * @description:
 * @param {*gin.Context} c
 * @param {string} Username
 * @return {*}
 * @method:
 * @router:
 */
func GetUserInfoByUsername(ctx echo.Context) error {
	username := ctx.Param("username")
	if len(username) == 0 {
		return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
	}
	user := service.MyService.User().GetUserInfoByUserName(username)
	if user.Id == 0 {
		return ctx.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.USER_NOT_EXIST, Message: common_err.GetMsg(common_err.USER_NOT_EXIST)})
	}

	return ctx.JSON(common_err.SUCCESS,
		model.Result{
			Success: common_err.SUCCESS,
			Message: common_err.GetMsg(common_err.SUCCESS),
			Data:    user,
		})
}

/**
 * @description: get all Usernames
 * @method:GET
 * @router:/user/all/name
 */
func GetUserAllUsername(ctx echo.Context) error {
	users := service.MyService.User().GetAllUserName()
	names := []string{}
	for _, v := range users {
		names = append(names, v.Username)
	}
	return ctx.JSON(common_err.SUCCESS,
		model.Result{
			Success: common_err.SUCCESS,
			Message: common_err.GetMsg(common_err.SUCCESS),
			Data:    names,
		})
}

/**
 * @description:get custom file by user
 * @param {path} name string "file name"
 * @method: GET
 * @router: /user/custom/:key
 */
func GetUserCustomConf(ctx echo.Context) error {
	name := ctx.Param("key")
	if len(name) == 0 {
		return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
	}
	id := ctx.Request().Header.Get("user_id")

	user := service.MyService.User().GetUserInfoById(id)
	//	user := service.MyService.User().GetUserInfoByUsername(Username)
	if user.Id == 0 {
		return ctx.JSON(common_err.SERVICE_ERROR,
			model.Result{Success: common_err.USER_NOT_EXIST, Message: common_err.GetMsg(common_err.USER_NOT_EXIST)})
	}
	filePath := config.AppInfo.UserDataPath + "/" + id + "/" + name + ".json"

	data := file.ReadFullFile(filePath)
	if !gjson.ValidBytes(data) {
		return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: string(data)})
	}
	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: json2.RawMessage(string(data))})
}

/**
 * @description:create or update custom conf by user
 * @param {path} name string "file name"
 * @method:POST
 * @router:/user/custom/:key
 */
func PostUserCustomConf(ctx echo.Context) error {
	name := ctx.Param("key")
	if len(name) == 0 {
		return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
	}
	id := ctx.Request().Header.Get("user_id")
	user := service.MyService.User().GetUserInfoById(id)
	if user.Id == 0 {
		return ctx.JSON(common_err.SERVICE_ERROR,
			model.Result{Success: common_err.USER_NOT_EXIST, Message: common_err.GetMsg(common_err.USER_NOT_EXIST)})
	}
	data, _ := io.ReadAll(ctx.Request().Body)
	filePath := config.AppInfo.UserDataPath + "/" + strconv.Itoa(user.Id)

	if err := file.IsNotExistMkDir(filePath); err != nil {
		return ctx.JSON(common_err.SERVICE_ERROR,
			model.Result{Success: common_err.SERVICE_ERROR, Message: common_err.GetMsg(common_err.SERVICE_ERROR)})
	}

	if err := file.WriteToPath(data, filePath, name+".json"); err != nil {
		return ctx.JSON(common_err.SERVICE_ERROR,
			model.Result{Success: common_err.SERVICE_ERROR, Message: common_err.GetMsg(common_err.SERVICE_ERROR)})
	}

	if name == "system" {
		dataMap := make(map[string]string, 1)
		dataMap["system"] = string(data)
		response, err := service.MyService.MessageBus().PublishEventWithResponse(context.Background(), common.SERVICENAME, "zimaos:user:save_config", dataMap)
		if err != nil {
			logger.Error("failed to publish event to message bus", zap.Error(err), zap.Any("event", string(data)))
		}
		if response.StatusCode() != http.StatusOK {
			logger.Error("failed to publish event to message bus", zap.String("status", response.Status()), zap.Any("response", response))
		}

	}

	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: json2.RawMessage(string(data))})
}

/**
 * @description: delete user custom config
 * @param {path} key string
 * @method:delete
 * @router:/user/custom/:key
 */
func DeleteUserCustomConf(ctx echo.Context) error {
	name := ctx.Param("key")
	if len(name) == 0 {
		return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
	}
	id := ctx.Request().Header.Get("user_id")
	user := service.MyService.User().GetUserInfoById(id)
	if user.Id == 0 {
		return ctx.JSON(common_err.SERVICE_ERROR,
			model.Result{Success: common_err.USER_NOT_EXIST, Message: common_err.GetMsg(common_err.USER_NOT_EXIST)})
	}
	filePath := config.AppInfo.UserDataPath + "/" + strconv.Itoa(user.Id) + "/" + name + ".json"
	err := os.Remove(filePath)
	if err != nil {
		return ctx.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.SERVICE_ERROR, Message: common_err.GetMsg(common_err.SERVICE_ERROR)})
	}
	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS)})
}

/**
 * @description:
 * @param {path} id string "user id"
 * @method:DELETE
 * @router:/user/delete/:id
 */
func DeleteUser(ctx echo.Context) error {
	id := ctx.Param("id")
	service.MyService.User().DeleteUserById(id)
	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: id})
}

/**
 * @description:update user image
 * @method:POST
 * @router:/user/current/image/:key
 */
func PutUserImage(ctx echo.Context) error {
	id := ctx.Request().Header.Get("user_id")
	json := make(map[string]string)
	ctx.Bind(&json)

	path := json["path"]
	key := ctx.Param("key")
	if len(path) == 0 || len(key) == 0 {
		return ctx.JSON(http.StatusOK, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
	}
	if !file.Exists(path) {
		return ctx.JSON(http.StatusOK, model.Result{Success: common_err.FILE_DOES_NOT_EXIST, Message: common_err.GetMsg(common_err.FILE_DOES_NOT_EXIST)})
	}

	_, err := file.GetImageExt(path)
	if err != nil {
		return ctx.JSON(http.StatusOK, model.Result{Success: common_err.NOT_IMAGE, Message: common_err.GetMsg(common_err.NOT_IMAGE)})
	}

	user := service.MyService.User().GetUserInfoById(id)
	if user.Id == 0 {
		return ctx.JSON(http.StatusOK, model.Result{Success: common_err.USER_NOT_EXIST, Message: common_err.GetMsg(common_err.USER_NOT_EXIST)})
	}
	fstat, _ := os.Stat(path)
	if fstat.Size() > 10<<20 {
		return ctx.JSON(http.StatusOK, model.Result{Success: common_err.IMAGE_TOO_LARGE, Message: common_err.GetMsg(common_err.IMAGE_TOO_LARGE)})
	}
	ext := file.GetExt(path)
	filePath := config.AppInfo.UserDataPath + "/" + strconv.Itoa(user.Id) + "/" + key + ext
	file.CopySingleFile(path, filePath, "overwrite")

	data := make(map[string]string, 3)
	data["path"] = filePath
	data["file_name"] = key + ext
	data["online_path"] = "/v1/users/image?path=" + filePath
	return ctx.JSON(http.StatusOK, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: data})
}

/**
* @description:
* @param {*gin.Context} c
* @param {file} file
* @param {string} key
* @param {string} type:avatar,background
* @return {*}
* @method:
* @router:
 */
func PostUserUploadImage(ctx echo.Context) error {
	id := ctx.Request().Header.Get("user_id")
	f, err := ctx.FormFile("file")
	key := ctx.Param("key")
	t := ctx.FormValue("type")
	if len(key) == 0 {
		return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
	}
	if err != nil {
		return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.CLIENT_ERROR, Message: common_err.GetMsg(common_err.CLIENT_ERROR), Data: err.Error()})
	}

	_, err = file.GetImageExtByName(f.Filename)
	if err != nil {
		return ctx.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.NOT_IMAGE, Message: common_err.GetMsg(common_err.NOT_IMAGE)})
	}
	ext := filepath.Ext(f.Filename)
	user := service.MyService.User().GetUserInfoById(id)

	if user.Id == 0 {
		return ctx.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.USER_NOT_EXIST, Message: common_err.GetMsg(common_err.USER_NOT_EXIST)})
	}
	if t == "avatar" {
		key = "avatar"
	}
	path := config.AppInfo.UserDataPath + "/" + strconv.Itoa(user.Id) + "/" + key + ext

	file.SaveUploadedFile(f, path)

	data := make(map[string]string, 3)
	data["path"] = path
	data["file_name"] = key + ext
	data["online_path"] = "/v1/users/image?path=" + path
	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: data})
}

/**
 * @description: get current user's image
 * @method:GET
 * @router:/user/image/:id
 */
func GetUserImage(ctx echo.Context) error {
	filePath := ctx.QueryParam("path")
	if len(filePath) == 0 {
		return ctx.JSON(http.StatusNotFound, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
	}
	absFilePath, err := filepath.Abs(filepath.Clean(filePath))
	if err != nil {
		return ctx.JSON(http.StatusNotFound, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
	}
	if !file.Exists(absFilePath) {
		return ctx.JSON(http.StatusNotFound, model.Result{Success: common_err.FILE_DOES_NOT_EXIST, Message: common_err.GetMsg(common_err.FILE_DOES_NOT_EXIST)})
	}
	if !strings.Contains(absFilePath, config.AppInfo.UserDataPath) {
		return ctx.JSON(http.StatusNotFound, model.Result{Success: common_err.INSUFFICIENT_PERMISSIONS, Message: common_err.GetMsg(common_err.INSUFFICIENT_PERMISSIONS)})
	}

	matched, err := regexp.MatchString(`^/var/lib/casaos/\d`, absFilePath)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, model.Result{Success: common_err.INSUFFICIENT_PERMISSIONS, Message: common_err.GetMsg(common_err.INSUFFICIENT_PERMISSIONS)})
	}
	if !matched {
		return ctx.JSON(http.StatusNotFound, model.Result{Success: common_err.INSUFFICIENT_PERMISSIONS, Message: common_err.GetMsg(common_err.INSUFFICIENT_PERMISSIONS)})
	}

	fileName := path.Base(absFilePath)

	// @tiger - RESTful 规范下不应该返回文件本身内容，而是返回文件的静态URL，由前端去解析
	ctx.Response().Header().Set("Content-Disposition", "attachment; filename*=utf-8''"+url2.PathEscape(fileName))
	return ctx.File(absFilePath)
}

func DeleteUserImage(ctx echo.Context) error {
	id := ctx.Request().Header.Get("user_id")
	path := ctx.QueryParam("path")
	if len(path) == 0 {
		return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
	}
	user := service.MyService.User().GetUserInfoById(id)
	if user.Id == 0 {
		return ctx.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.USER_NOT_EXIST, Message: common_err.GetMsg(common_err.USER_NOT_EXIST)})
	}
	if !file.Exists(path) {
		return ctx.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.FILE_DOES_NOT_EXIST, Message: common_err.GetMsg(common_err.FILE_DOES_NOT_EXIST)})
	}
	if !strings.Contains(path, config.AppInfo.UserDataPath+"/"+strconv.Itoa(user.Id)) {
		return ctx.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.INSUFFICIENT_PERMISSIONS, Message: common_err.GetMsg(common_err.INSUFFICIENT_PERMISSIONS)})
	}
	os.Remove(path)
	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS)})
}

/**
 * @description:
 * @param {*gin.Context} c
 * @param {string} refresh_token
 * @return {*}
 * @method:
 * @router:
 */
func PostUserRefreshToken(ctx echo.Context) error {
	js := make(map[string]string)
	ctx.Bind(&js)
	refresh := js["refresh_token"]

	privateKey, _ := service.MyService.User().GetKeyPair()

	claims, err := jwt.ParseToken(
		refresh,
		func() (*ecdsa.PublicKey, error) {
			_, publicKey := service.MyService.User().GetKeyPair()
			return publicKey, nil
		})
	if err != nil {
		return ctx.JSON(http.StatusUnauthorized, model.Result{Success: common_err.VERIFICATION_FAILURE, Message: common_err.GetMsg(common_err.VERIFICATION_FAILURE), Data: err.Error()})
	}
	if !claims.VerifyExpiresAt(time.Now(), true) || !claims.VerifyIssuer("refresh", true) {
		return ctx.JSON(http.StatusUnauthorized, model.Result{Success: common_err.VERIFICATION_FAILURE, Message: common_err.GetMsg(common_err.VERIFICATION_FAILURE)})
	}

	newAccessToken, err := jwt.GetAccessToken(claims.Username, privateKey, claims.ID)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, model.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
	}

	newRefreshToken, err := jwt.GetRefreshToken(claims.Username, privateKey, claims.ID)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, model.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
	}

	verifyInfo := system_model.VerifyInformation{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    time.Now().Add(3 * time.Hour).Unix(),
	}

	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: verifyInfo})
}

func DeleteUserAll(ctx echo.Context) error {
	service.MyService.User().DeleteAllUser()
	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS)})
}

// @Summary 检查是否进入引导状态
// @Produce  application/json
// @Accept application/json
// @Tags sys
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /sys/init/check [get]
func GetUserStatus(ctx echo.Context) error {
	data := make(map[string]interface{}, 2)

	if service.MyService.User().GetUserCount() > 0 {
		data["initialized"] = true
		data["key"] = ""
	} else {
		key := uuid.NewV4().String()
		service.UserRegisterHash[key] = key
		data["key"] = key
		data["initialized"] = false
	}
	gpus, err := external.NvidiaGPUInfoList()
	if err != nil {
		logger.Error("NvidiaGPUInfoList error", zap.Error(err))
	}
	data["gpus"] = len(gpus)
	return ctx.JSON(common_err.SUCCESS,
		model.Result{
			Success: common_err.SUCCESS,
			Message: common_err.GetMsg(common_err.SUCCESS),
			Data:    data,
		})
}
