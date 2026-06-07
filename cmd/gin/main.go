package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type DataResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type DataItem struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}

var mockData = []DataItem{
	{ID: 1, Name: "item1", Value: "value1", CreatedAt: time.Now()},
	{ID: 2, Name: "item2", Value: "value2", CreatedAt: time.Now()},
	{ID: 3, Name: "item3", Value: "value3", CreatedAt: time.Now()},
}

func main() {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	api := r.Group("/api/v1")
	{
		api.GET("/data", GetDataHandler)
		api.GET("/data/:id", GetDataByIDHandler)
		api.POST("/data", CreateDataHandler)
	}

	r.Run(":8080")
}

func GetDataHandler(c *gin.Context) {
	c.JSON(http.StatusOK, DataResponse{
		Code:    200,
		Message: "success",
		Data:    mockData,
	})
}

func GetDataByIDHandler(c *gin.Context) {
	id := c.Param("id")

	for _, item := range mockData {
		if item.ID == 1 && id == "1" ||
			item.ID == 2 && id == "2" ||
			item.ID == 3 && id == "3" {
			c.JSON(http.StatusOK, DataResponse{
				Code:    200,
				Message: "success",
				Data:    item,
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, DataResponse{
		Code:    404,
		Message: "data not found",
		Data:    nil,
	})
}

func CreateDataHandler(c *gin.Context) {
	var newItem DataItem
	if err := c.ShouldBindJSON(&newItem); err != nil {
		c.JSON(http.StatusBadRequest, DataResponse{
			Code:    400,
			Message: "invalid request: " + err.Error(),
			Data:    nil,
		})
		return
	}

	newItem.ID = len(mockData) + 1
	newItem.CreatedAt = time.Now()
	mockData = append(mockData, newItem)

	c.JSON(http.StatusCreated, DataResponse{
		Code:    201,
		Message: "data created successfully",
		Data:    newItem,
	})
}
