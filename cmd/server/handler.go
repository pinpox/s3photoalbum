package main

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"net/http"
)

func login(c *gin.Context) {

	formUser := c.PostForm("username")
	formPass := c.PostForm("password")

	td := templateData{
		Context: c,
		Data: struct {
			Title string
			Error string
		}{
			Title: "Login",
			Error: "Authentication failed",
		},
	}

	user, err := findUserByUsername(formUser)
	if err != nil {
		// User not found
		log.Warn("User not found", err)
		c.HTML(http.StatusOK, "login.html", td)
		c.Abort()
		return
	}

	// Comparing the password with the hash
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(formPass)); err != nil {
		log.Warn("Invalid password", err)
		c.HTML(http.StatusOK, "login.html", td)
		c.Abort()
		return
	}

	token, err := generateToken(*user)
	if err != nil {
		log.Warn("token error", err)
		c.HTML(http.StatusOK, "login.html", td)
		c.Abort()
		return
	}

	// func (c *Context) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool)
	// TODO check parameters
	// TODO refersh the token before it expires
	c.SetCookie("token", token, 3600, "/", envHost, true, false)

	log.Infof("User %s logged in, redirecting to /\n", formUser)
	c.Redirect(http.StatusSeeOther, "/")

}

func getUserInfo(c *gin.Context) {
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
