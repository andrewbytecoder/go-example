package api

import (
	"github.com/zzu-andrew/go-example/gin_master/gin"
	"net/http"
)

type DataApi struct {
}

// GetData godoc
// @Summary Get Data
// @Description Get data from the server
// @Tags data
// @Accept json
// @Produce json
// @Success 200 {object} gin.H
// @Router /data [get]
func (dataApi *DataApi) GetData(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

// PostData godoc
// @Summary Post Data
// @Description Post data to the server
// @Tags data
// @Accept json
// @Produce json
// @Success 200 {object} gin.H
// @Router /data [post]
func (dataApi *DataApi) PostData(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

// PutData godoc
// @Summary Put Data
// @Description Update data on the server
// @Tags data
// @Accept json
// @Produce json
// @Success 200 {object} gin.H
// @Router /data [put]
func (dataApi *DataApi) PutData(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

// DeleteData godoc
// @Summary Delete Data
// @Description Delete data from the server
// @Tags data
// @Accept json
// @Produce json
// @Success 200 {object} gin.H
// @Router /data [delete]
func (dataApi *DataApi) DeleteData(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}
