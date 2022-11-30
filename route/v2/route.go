package v2

import codegen "github.com/IceWhaleTech/CasaOS-UserService/codegen/user_service"

type UserService struct{}

func NewUserService() codegen.ServerInterface {
	return &UserService{}
}
