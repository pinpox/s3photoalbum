package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username string `json:"username"`
	Password string `json:"password"`
	IsAdmin  bool   `json:"isadmin"`
	Age      uint   `json:"age"`
}

func insertUser(username string, password string, isadmin bool, age uint) (*User, error) {
	user := User{
		Username: username,
		Password: password,
		IsAdmin:  isadmin,
		Age:      age,
	}
	if res := DB.Create(&user); res.Error != nil {
		return nil, res.Error
	}
	return &user, nil
}

func findUserByUsername(username string) (*User, error) {
	var user User
	if res := DB.Where("username = ?", username).Find(&user); res.Error != nil {
		return nil, res.Error
	}
	return &user, nil
}

func findUserByID(id uint) (*User, error) {
	var user User
	if res := DB.Find(&user, id); res.Error != nil {
		return nil, res.Error
	}
	return &user, nil
}

func getSession(c *gin.Context) (uint, string, bool) {

	fmt.Println("getting session")
	id, ok := c.Get("id")
	if !ok {
		return 0, "", false
	}
	username, ok := c.Get("username")
	if !ok {
		return 0, "", false
	}
	return id.(uint), username.(string), true
}
