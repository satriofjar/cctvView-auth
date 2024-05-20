package main

import (
	"cctvView/models"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{})
}

func LoginHandler(c *gin.Context) {
	UserName := c.PostForm("username")
	password := c.PostForm("password")

	var user models.User

	log.Println("username : ", UserName)
	log.Println("password : ", password)

	if err := models.DB.Where("user_name = ?", UserName).First(&user).Error; err != nil {
		switch err {
		case gorm.ErrRecordNotFound:
			c.HTML(http.StatusNotFound, "login.html", gin.H{"message": "Username incorrect"})
			// c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"message": "Username not found"})
			return

		default:
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"messagenih": err.Error()})
			return
		}
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.HashPassword), []byte(password))

	if err == nil {
		session, _ := store.Get(c.Request, "session")
		// session struct has field Values map[interface{}]interface{}
		session.Values["user"] = UserName
		// save before writing to response/return from handler
		session.Save(c.Request, c.Writer)
		c.Redirect(http.StatusMovedPermanently, "/stream/floor/lantai_1")
	}

}

func RegisterPage(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", gin.H{})
}

func RegisterHandler(c *gin.Context) {
	// var user models.User
	userName := c.PostForm("username")
	password := c.PostForm("password")
	confirmPassword := c.PostForm("conpassword")

	log.Println("username : ", userName)
	log.Println("password : ", password)
	log.Println("confirm password : ", confirmPassword)

	// Check if passwords match
	if password != confirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password and Confirm Password do not match"})
		return
	}

	// Check if the username already exists
	var existingUser models.User

	if err := models.DB.Where("user_name = ?", userName).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username already exists"})
		return
	}

	// create hash from password
	var hash []byte
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("bcrypt err:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	// Create the user in the database
	user := models.User{
		UserName:     userName,
		HashPassword: string(hash),
	}

	log.Println(user)

	if err := models.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.HTML(http.StatusOK, "login.html", gin.H{})
}

func LogoutHandler(c *gin.Context) {
	session, _ := store.Get(c.Request, "session")
	delete(session.Values, "user")
	session.Save(c.Request, c.Writer)
	c.HTML(http.StatusOK, "login.html", gin.H{"message": "Logged out"})
}
