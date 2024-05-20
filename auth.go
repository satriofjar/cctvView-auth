package main

import (
	"encoding/gob"
	"net/http"

	"cctvView/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

var store = sessions.NewCookieStore([]byte("super-secret"))

func init() {
	var user models.User
	store.Options.HttpOnly = true // since we are not accessing any cookies w/ JavaScript, set to true
	store.Options.Secure = true   // requires secuire HTTPS connection
	// Set MaxAge to 24 hours (86400 seconds)
	store.Options.MaxAge = 86400 // 24 hours in seconds
	gob.Register(&user)
}

// auth middleware checks if logged in by looking at session
func auth(c *gin.Context) {
	session, _ := store.Get(c.Request, "session")
	_, ok := session.Values["user"]
	if !ok {
		c.HTML(http.StatusForbidden, "login.html", nil)
		c.Abort()
		return
	}
	c.Next()
}
