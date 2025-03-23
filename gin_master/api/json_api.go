package api

import (
	"github.com/zzu-andrew/go-example/gin_master/gin"
	"net/http"
)

type JsonApi struct {
}

func (j *JsonApi) GetJson(c *gin.Context) {
	// 定义嵌套的 JSON 数据
	nestedJSON := gin.H{
		"status": "success",
		"data": gin.H{
			"user": gin.H{
				"id":    1,
				"name":  "Alice",
				"email": "alice@example.com",
			},
			"profile": gin.H{
				"age":     25,
				"address": "123 Wonderland Ave",
				"hobbies": []string{"reading", "swimming", "coding"},
			},
		},
	}

	c.JSON(http.StatusOK, nestedJSON)
}

type Student struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (j *JsonApi) GetStructJson(c *gin.Context) {
	student := Student{
		ID:    1,
		Name:  "Alice",
		Email: "alice@example.com",
	}
	c.JSON(http.StatusOK, &student)
}

func (j *JsonApi) PostJson(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}

func (j *JsonApi) PutJson(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}

func (j *JsonApi) DeleteJson(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}
