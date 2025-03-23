package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httputil"
)

func main() {
	router := gin.Default()

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
	router.Any("/proxy", func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	})

	// 启动服务在192.168.0.1:8080
	router.Run("192.168.0.1:8080")
}
