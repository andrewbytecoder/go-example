package api

import (
	"github.com/zzu-andrew/go-example/gin_master/gin"
	"net/http"
)

type QueryApi struct {
}

// http://127.0.0.1:8080/query/name?name=andrew

// GetQuery godoc
// @Summary Get Query
// @Description Get query parameters from the server
// @Tags query
// @Accept json
// @Produce json
// @Param name query string true "Name of the user"
// @Param id query string false "ID of the user" default(1234)
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Router /query [get]
func (q *QueryApi) GetQuery(c *gin.Context) {
	// 没有默认值的查询
	//name := c.Query("name")
	name, ok := c.GetQuery("name")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"name": "",
		})
		return
	}
	// 默认值的查询
	id := c.DefaultQuery("id", "1234")
	c.JSON(http.StatusOK, gin.H{
		"name": name,
		"id":   id,
	})
}

// PutQuery godoc
// @Summary Put Query
// @Description Update query data on the server
// @Tags query
// @Accept json
// @Produce json
// @Param name formData string true "Name of the user"
// @Success 200 {object} gin.H
// @Router /query [put]
func (q *QueryApi) PutQuery(c *gin.Context) {
	name := c.PostForm("name")
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
		"name":    name,
	})
}

// DeleteQuery godoc
// @Summary Delete Query
// @Description Delete query data from the server
// @Tags query
// @Accept json
// @Produce json
// @Param name formData string true "Name of the user"
// @Success 200 {object} gin.H
// @Router /query [delete]
func (q *QueryApi) DeleteQuery(c *gin.Context) {
	name := c.PostForm("name")
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
		"name":    name,
	})
}
