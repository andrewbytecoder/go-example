package api

import (
	"github.com/zzu-andrew/go-example/gin_master/gin"
)

var apiRouter = new(IRouter)

type IRouter struct {
	DataApi
	FormApi
	QueryApi
	ReserveApi
	JsonApi
	UrlApi
}

func Router() *gin.Engine {

	router := gin.Default()

	//router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	dataApi := router.Group("/data")
	{
		dataApi.GET("/data", apiRouter.GetData)
		dataApi.POST("/data", apiRouter.PostData)
		dataApi.DELETE("/data", apiRouter.DeleteData)
		dataApi.PUT("/data", apiRouter.PutData)
	}

	reserveApi := router.Group("/reserve")
	{
		reserveApi.GET("/proxy", apiRouter.GetReserve)
		reserveApi.POST("/proxy", apiRouter.PostReserve)
		reserveApi.DELETE("/proxy", apiRouter.DeleteReserve)
		reserveApi.PUT("/proxy", apiRouter.PutReserve)
	}

	jsonAPi := router.Group("/json")
	{
		jsonAPi.GET("/string", apiRouter.GetJson)
		jsonAPi.GET("/struct", apiRouter.GetStructJson)
		jsonAPi.POST("/data", apiRouter.PostJson)
		jsonAPi.DELETE("/data", apiRouter.DeleteJson)
		jsonAPi.PUT("/data", apiRouter.PutJson)
	}

	queryApi := router.Group("/query")
	{
		queryApi.GET("/name", apiRouter.GetQuery)
		queryApi.DELETE("/query", apiRouter.DeleteQuery)
		queryApi.PUT("/query", apiRouter.PutQuery)
	}

	formApi := router.Group("/form")
	{
		formApi.GET("/get", apiRouter.GetForm)
		formApi.POST("/post", apiRouter.PostForm)
		formApi.DELETE("/delete", apiRouter.DeleteForm)
		formApi.PUT("/put", apiRouter.PutForm)
	}

	urlApi := router.Group("/url")
	{
		urlApi.GET("/:name/:age", apiRouter.GetUrl)
		urlApi.POST("/url", apiRouter.PostUrl)
		urlApi.DELETE("/url", apiRouter.DeleteUrl)
		urlApi.PUT("/url", apiRouter.PutUrl)
	}

	router.Run(":8080")
	// 正常走不到这里
	return router
}
