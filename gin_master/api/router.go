package api

import "github.com/zzu-andrew/github.com/zzu-andrew/go-example/gin_master/gin"

var apiRouter = new(IRouter)

type IRouter struct {
	DataApi
	FormApi
	QueryApi
}

func Router() *gin.Engine {

	router := gin.Default()

	dataApi := router.Group("/api")

	{
		dataApi.GET("/data", apiRouter.GetData)
		dataApi.POST("/data", apiRouter.PostData)
		dataApi.DELETE("/data", apiRouter.DeleteData)
		dataApi.PUT("/data", apiRouter.PutData)
	}

	router.GET("/json", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"html": "<b>Hello, world!</b>",
		})
	})

	router.Run(":8080")
	// 正常走不到这里
	return router
}
