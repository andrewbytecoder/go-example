package main

import "go-example/gin_master/gin"

//go:generate swag init --parseDependency --parseDepth=6 --instanceName admin -o ./doc/admin
func main() {
	router := gin.Default()

	router.GET("/json", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"html": "<b>Hello, world!</b>",
		})
	})

	router.Run(":8080")
}
