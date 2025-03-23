package api

import (
	"github.com/zzu-andrew/go-example/gin_master/gin"
	"net/http"
)

type RedirectApi struct {
}

// GetRedirect godoc
// @Summary Get Redirect
// @Description Redirect to Baidu using GET method
// @Tags redirect
// @Accept json
// @Produce json
// @Success 301 {string} string "Redirect to https://www.baidu.com"
// @Router /redirect [get]
func (r *RedirectApi) GetRedirect(c *gin.Context) {
	c.Redirect(http.StatusMovedPermanently, "https://www.baidu.com")
}

// PostRedirect godoc
// @Summary Post Redirect
// @Description Redirect to Baidu using POST method
// @Tags redirect
// @Accept json
// @Produce json
// @Success 301 {string} string "Redirect to https://www.baidu.com"
// @Router /redirect [post]
func (r *RedirectApi) PostRedirect(c *gin.Context) {
	c.Redirect(http.StatusMovedPermanently, "https://www.baidu.com")
}

// PutRedirect godoc
// @Summary Put Redirect
// @Description Redirect to Baidu using PUT method
// @Tags redirect
// @Accept json
// @Produce json
// @Success 301 {string} string "Redirect to https://www.baidu.com"
// @Router /redirect [put]
func (r *RedirectApi) PutRedirect(c *gin.Context) {
	c.Redirect(http.StatusMovedPermanently, "https://www.baidu.com")
}

// DeleteRedirect godoc
// @Summary Delete Redirect
// @Description Redirect to Baidu using DELETE method
// @Tags redirect
// @Accept json
// @Produce json
// @Success 301 {string} string "Redirect to https://www.baidu.com"
// @Router /redirect [delete]
func (r *RedirectApi) DeleteRedirect(c *gin.Context) {
	c.Redirect(http.StatusMovedPermanently, "https://www.baidu.com")
}
