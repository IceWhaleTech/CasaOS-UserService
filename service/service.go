package service

var MyService Repository

type Repository interface {
	User() UserService
}
