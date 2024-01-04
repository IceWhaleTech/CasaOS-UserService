package route

import (
	"crypto/ecdsa"
	"os"

	"github.com/IceWhaleTech/CasaOS-Common/middleware"
	"github.com/IceWhaleTech/CasaOS-Common/utils/jwt"
	v1 "github.com/IceWhaleTech/CasaOS-UserService/route/v1"
	"github.com/IceWhaleTech/CasaOS-UserService/service"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	r := gin.Default()
	r.Use(middleware.Cors())
	// r.Use(middleware.WriteLog())
	r.Use(gzip.Gzip(gzip.DefaultCompression))

	// check if environment variable is set
	if ginMode, success := os.LookupEnv("GIN_MODE"); success {
		gin.SetMode(ginMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r.POST("/v1/users/register", v1.PostUserRegister)
	r.POST("/v1/users/login", v1.PostUserLogin)
	r.GET("/v1/users/name", v1.GetUserAllUsername) // all/name
	r.POST("/v1/users/refresh", v1.PostUserRefreshToken)
	// No short-term modifications
	r.GET("/v1/users/image", v1.GetUserImage)

	r.GET("/v1/users/status", v1.GetUserStatus) // init/check

	v1Group := r.Group("/v1")

	v1Group.Use(jwt.JWT(
		func() (*ecdsa.PublicKey, error) {
			_, publicKey := service.MyService.User().GetKeyPair()
			return publicKey, nil
		},
	))
	{
		v1UsersGroup := v1Group.Group("/users")
		v1UsersGroup.Use()
		{
			v1UsersGroup.GET("/current", v1.GetUserInfo)
			v1UsersGroup.PUT("/current", v1.PutUserInfo)
			v1UsersGroup.PUT("/current/password", v1.PutUserPassword)

			v1UsersGroup.GET("/current/custom/:key", v1.GetUserCustomConf)
			v1UsersGroup.POST("/current/custom/:key", v1.PostUserCustomConf)
			v1UsersGroup.DELETE("/current/custom/:key", v1.DeleteUserCustomConf)

			v1UsersGroup.POST("/current/image/:key", v1.PostUserUploadImage)
			v1UsersGroup.PUT("/current/image/:key", v1.PutUserImage)
			// v1UserGroup.POST("/file/image/:key", v1.PostUserFileImage)
			v1UsersGroup.DELETE("/current/image", v1.DeleteUserImage)

			v1UsersGroup.PUT("/avatar", v1.PutUserAvatar)
			v1UsersGroup.GET("/avatar", v1.GetUserAvatar)

			v1UsersGroup.DELETE("/:id", v1.DeleteUser)
			v1UsersGroup.GET("/:username", v1.GetUserInfoByUsername)
			v1UsersGroup.DELETE("", v1.DeleteUserAll)
		}
	}

	return r
}
