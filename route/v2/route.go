package v2

import codegen "github.com/IceWhaleTech/CasaOS-UserService/codegen/user-service"

type UserService struct{}

func NewUserService() codegen.ServerInterface {
	return &UserService{}
}
