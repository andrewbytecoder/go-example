package api

import (
	"github.com/zzu-andrew/go-example/gin_master/gin"
	"net/http"
	"net/http/httputil"
)

type ReserveApi struct {
}

// GetReserve godoc
// @Summary Get Reserve
// @Description Get reserve data from the server
// @Tags reserve
// @Accept json
// @Produce json
// @Success 200 {object} gin.H
// @Router /reserve [get]
func (r *ReserveApi) GetReserve(c *gin.Context) {

	// 创建反向代理
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			// 修改请求的Scheme和Host为目标地址
			req.URL.Scheme = "http"
			req.URL.Host = "127.0.0.1:9000"
			// 设置Host头，确保目标服务器正确接收
			req.Host = "127.0.0.1:9000"
			// 保留原始路径和查询参数，无需修改
		},
	}
	// 处理所有HTTP方法的/json路径请求
	proxy.ServeHTTP(c.Writer, c.Request)
}

// PostReserve godoc
// @Summary Post Reserve
// @Description Post reserve data to the server
// @Tags reserve
// @Accept json
// @Produce json
// @Success 200 {object} gin.H
// @Router /reserve [post]
func (r *ReserveApi) PostReserve(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "ok",
		"data": "reserve",
	})
}

// PutReserve godoc
// @Summary Put Reserve
// @Description Update reserve data on the server
// @Tags reserve
// @Accept json
// @Produce json
// @Success 200 {object} gin.H
// @Router /reserve [put]
func (r *ReserveApi) PutReserve(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "ok",
		"data": "reserve",
	})
}

// DeleteReserve godoc
// @Summary Delete Reserve
// @Description Delete reserve data from the server
// @Tags reserve
// @Accept json
// @Produce json
// @Success 200 {object} gin.H
// @Router /reserve [delete]
func (r *ReserveApi) DeleteReserve(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "ok",
		"data": "reserve",
	})
}
