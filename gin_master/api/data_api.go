package api

import (
	"github.com/zzu-andrew/github.com/zzu-andrew/go-example/gin_master/gin"
	"net/http"
)

type DataApi struct {
}

func (dataApi *DataApi) GetData(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func (dataApi *DataApi) PostData(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func (dataApi *DataApi) PutData(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func (dataApi *DataApi) DeleteData(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}
