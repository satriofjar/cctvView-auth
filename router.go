package main

import (
	"cctvView/models"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func serveHTTP() {
	router := gin.Default()
	models.ConnectDatabase()
	router.LoadHTMLGlob("web/templates/*")

	// Authentication router not protected
	router.GET("/login", LoginPage)
	router.POST("/login-post", LoginHandler)
	router.GET("/register", RegisterPage)
	router.POST("/register-post", RegisterHandler)

	// protected router
	authRoute := router.Group("", auth)
	authRoute.GET("/", HTTPAPIServerIndex)
	authRoute.GET("/stream/floor/:uuid", HTTPAPIServerFloor)
	authRoute.GET("/stream/player/:uuid", HTTPAPIServerStreamPlayer)
	authRoute.GET("/stream/thumbnail/:uuid", HTTPAPIServerThumbnail)
	authRoute.POST("/stream/receiver/:uuid", HTTPAPIServerStreamWebRTC)
	authRoute.GET("/stream/codec/:uuid", HTTPAPIServerStreamCodec)
	authRoute.GET("/logout", LogoutHandler)

	router.StaticFS("/static", http.Dir("web/static"))
	err := router.Run(Config.Server.HTTPPort)
	if err != nil {
		log.Fatalln("Start HTTP Server error", err)
	}
}
