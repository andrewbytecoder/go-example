package api

import (
	"github.com/zzu-andrew/go-example/gin_master/gin"
	"net/http"
)

type FormApi struct {
}

// GetForm godoc
// @Summary Get Form
// @Description Get form data from the server
// @Tags form
// @Accept json
// @Produce json
// @Success 200 {object} gin.H
// @Router /form [get]
func (formApi *FormApi) GetForm(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "get form",
	})
}

// PostForm godoc
// @Summary Post Form
// @Description Post form data to the server
// @Tags form
// @Accept json
// @Produce json
// @Success 200 {object} gin.H
// @Router /form [post]
func (formApi *FormApi) PostForm(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "post form",
	})
}

// PutForm godoc
// @Summary Put Form
// @Description Update form data on the server
// @Tags form
// @Accept json
// @Produce json
// @Success 200 {object} gin.H
// @Router /form [put]
func (formApi *FormApi) PutForm(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "put form",
	})
}

// DeleteForm godoc
// @Summary Delete Form
// @Description Delete form data from the server
// @Tags form
// @Accept json
// @Produce json
// @Success 200 {object} gin.H
// @Router /form [delete]
func (formApi *FormApi) DeleteForm(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "delete form",
	})
}
