package api

import (
	"github.com/zzu-andrew/github.com/zzu-andrew/go-example/gin_master/gin"
	"net/http"
)

type FormApi struct {
}

func (formApi *FormApi) GetForm(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "get form",
	})
}

func (formApi *FormApi) PostForm(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "post form",
	})
}

func (formApi *FormApi) PutForm(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "put form",
	})
}
func (formApi *FormApi) DeleteForm(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "delete form",
	})
}
