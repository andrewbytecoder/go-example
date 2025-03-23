package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func main() {
	// Create a default Gin engine
	r := gin.Default()

	// Load static files
	r.Static("gin_master/template/static", "./gin_master/template/static")

	// Load HTML templates
	r.LoadHTMLGlob("gin_master/template/static/*")

	// Define routes
	r.GET("/changePassword", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	r.POST("/user/changePassword", func(c *gin.Context) {
		password := c.PostForm("password")
		newPassword := c.PostForm("newPassword")

		// Here you would typically validate the password and update the database
		// For simplicity, we'll just log the values
		c.JSON(http.StatusOK, gin.H{
			"message":     "Password changed successfully",
			"password":    password,
			"newPassword": newPassword,
		})
	})

	// Run the server
	r.Run(":9090")
}
