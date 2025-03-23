package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func main() {
	r := gin.Default()

	// 设置路由处理登录请求
	r.POST("/login", func(c *gin.Context) {
		username := c.PostForm("username")
		password := c.PostForm("password")

		// 这里只是一个简单的验证示例
		if username == "admin" && password == "admin" {
			c.JSON(http.StatusOK, gin.H{
				"status":  "success",
				"message": "Login successful",
			})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "Invalid username or password",
			})
		}
	})
	// 设置路由处理静态文件（HTML）
	r.LoadHTMLGlob("templates/*")

	r.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})

	// 启动服务器，默认监听在 0.0.0.0:8080
	r.Run()
}
