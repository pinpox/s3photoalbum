package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func login(c *gin.Context) {

	formUser := c.PostForm("username")
	formPass := c.PostForm("password")

	user, err := findUserByUsername(formUser)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("user %s not found", formUser),
		})
		return
	}

	fmt.Println(user)

	if user.Password != formPass {
		c.HTML(http.StatusOK, "login.html", "Incorrect password")
		return
	}
	token, err := generateToken(*user)
	if err != nil {
		c.HTML(http.StatusOK, "login.html", "Authentication failed")
		return
	}

	// func (c *Context) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool)
	// TODO check parameters
	// TODO refersh the token before it expires
	c.SetCookie("token", token, 3600, "/", "localhost", true, false)

	c.Redirect(http.StatusSeeOther, "/")

}

func getUserInfo(c *gin.Context) {
	fmt.Println("getting session")
	id, _, ok := getSession(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{})
		return
	}
	user, err := findUserByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{})
		return
	}
	c.JSON(http.StatusOK, user)
}
