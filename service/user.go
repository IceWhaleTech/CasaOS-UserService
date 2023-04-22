/*
 * @Author: LinkLeong link@icewhale.com
 * @Date: 2022-03-18 11:40:55
 * @LastEditors: LinkLeong
 * @LastEditTime: 2022-07-12 10:05:37
 * @Description:
 * @Website: https://www.casaos.io
 * Copyright (c) 2022 by icewhale, All Rights Reserved.
 */
package service

import (
	"crypto/ecdsa"
	"io"
	"mime/multipart"
	"os"

	"github.com/IceWhaleTech/CasaOS-Common/utils/jwt"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-UserService/service/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserService interface {
	UpLoadFile(file multipart.File, name string) error
	CreateUser(m model.UserDBModel) model.UserDBModel
	GetUserCount() (userCount int64)
	UpdateUser(m model.UserDBModel)
	UpdateUserPassword(m model.UserDBModel)
	GetUserInfoById(id string) (m model.UserDBModel)
	GetUserAllInfoById(id string) (m model.UserDBModel)
	GetUserAllInfoByName(userName string) (m model.UserDBModel)
	DeleteUserById(id string)
	DeleteAllUser()
	GetUserInfoByUserName(userName string) (m model.UserDBModel)
	GetAllUserName() (list []model.UserDBModel)

	GetKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey)
}

var UserRegisterHash = make(map[string]string)

type userService struct {
	privateKey *ecdsa.PrivateKey // keep this private - NEVER expose it!!!
	publicKey  *ecdsa.PublicKey

	db *gorm.DB
}

func (u *userService) DeleteAllUser() {
	u.db.Where("1=1").Delete(&model.UserDBModel{})
}

func (u *userService) DeleteUserById(id string) {
	u.db.Where("id= ?", id).Delete(&model.UserDBModel{})
}

func (u *userService) GetAllUserName() (list []model.UserDBModel) {
	u.db.Select("username").Find(&list)
	return
}

func (u *userService) CreateUser(m model.UserDBModel) model.UserDBModel {
	u.db.Create(&m)
	return m
}

func (u *userService) GetUserCount() (userCount int64) {
	u.db.Find(&model.UserDBModel{}).Count(&userCount)
	return
}

func (u *userService) UpdateUser(m model.UserDBModel) {
	u.db.Model(&m).Omit("password").Updates(&m)
}

func (u *userService) UpdateUserPassword(m model.UserDBModel) {
	u.db.Model(&m).Update("password", m.Password)
}

func (u *userService) GetUserAllInfoById(id string) (m model.UserDBModel) {
	u.db.Where("id= ?", id).First(&m)
	return
}

func (u *userService) GetUserAllInfoByName(userName string) (m model.UserDBModel) {
	u.db.Where("username= ?", userName).First(&m)
	return
}

func (u *userService) GetUserInfoById(id string) (m model.UserDBModel) {
	u.db.Select("username", "id", "role", "nickname", "description", "avatar", "email").Where("id= ?", id).First(&m)
	return
}

func (u *userService) GetUserInfoByUserName(userName string) (m model.UserDBModel) {
	u.db.Select("username", "id", "role", "nickname", "description", "avatar", "email").Where("username= ?", userName).First(&m)
	return
}

// 上传文件
func (c *userService) UpLoadFile(file multipart.File, url string) error {
	out, _ := os.OpenFile(url, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0o644)
	defer out.Close()
	io.Copy(out, file)
	return nil
}

func (u *userService) GetKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	return u.privateKey, u.publicKey
}

// 获取用户Service
func NewUserService(db *gorm.DB) UserService {
	// DO NOT store private key anywhere - keep it in memory ONLY!!!
	privateKey, publicKey, err := jwt.GenerateKeyPair()
	if err != nil {
		logger.Error("failed to generate key pair for JWT", zap.Error(err))
		return nil
	}

	return &userService{
		privateKey: privateKey,
		publicKey:  publicKey,
		db:         db,
	}
}
