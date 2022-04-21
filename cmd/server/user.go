package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"strconv"

	"gorm.io/gorm/clause"
)

type User struct {
	gorm.Model
	Username string `gorm:"unique;not null"`
	Password string `gorm:"not null"`
	IsAdmin  bool   `gorm:"not null;default:false"`
	Age      uint
}

func insertUser(username string, password string, isadmin bool, age uint) (*User, error) {
	user := User{
		Username: username,
		Password: password,
		IsAdmin:  isadmin,
		Age:      age,
	}
	if res := DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&user); res.Error != nil {
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

func deleteUser(c *gin.Context) {
	formUser := c.Param("user")

	result := DB.Delete(&User{}, formUser)
	if result.Error != nil {
		fmt.Println(result.Error)
	}

	c.Redirect(http.StatusSeeOther, "/users")
}

func createUser(c *gin.Context) {

	formUser := c.PostForm("username")
	formPass := c.PostForm("password")
	formIsAdmin := c.PostForm("isadmin")
	formAge := c.PostForm("age")

	passwordHash, err := hashAndSalt(formPass)
	if err != nil {
		fmt.Println("failed to hash pass", err)
		getUsers(c)
	}

	userAge, err := strconv.ParseUint(formAge, 10, 64)
	if err != nil {
		fmt.Println("failed to convert age", err)
		getUsers(c)
	}

	_, err = insertUser(formUser, passwordHash, formIsAdmin == "on", uint(userAge))
	if err != nil {
		fmt.Println("failed to insert user", err)
		getUsers(c)
	}

	c.Redirect(http.StatusSeeOther, "/users")

}

func getUsers(c *gin.Context) {

	var users []User

	result := DB.Find(&users)
	if result.Error != nil {
		panic(result.Error)
	}

	td := templateData{
		Context: c,
		Data:    users,
	}

	c.HTML(http.StatusOK, "users.html", td)
}
