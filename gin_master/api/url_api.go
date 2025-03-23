package api

import (
	"github.com/zzu-andrew/go-example/gin_master/gin"
	"net/http"
)

type UrlApi struct {
}

func (urlApi *UrlApi) GetUrl(c *gin.Context) {
	name := c.Param("name")
	age := c.Param("age")
	c.JSON(http.StatusOK, gin.H{
		"name": name,
		"age":  age,
	})
}

func (urlApi *UrlApi) PostUrl(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func (urlApi *UrlApi) PutUrl(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}
func (urlApi *UrlApi) DeleteUrl(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}
