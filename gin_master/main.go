package main

import "github.com/zzu-andrew/go-example/gin_master/gin"

func main() {
	router := gin.Default()

	router.GET("/json", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"html": "<b>Hello, world!</b>",
		})
	})

	router.Run(":8080")
}
