package main

import (
	"fmt"
	"net/http"
)

type Context struct {
	request  *http.Request
	w        http.ResponseWriter
	index    int
	handlers []HandlerFun
}

func (c *Context) Next() {
	c.index++
	if c.index < len(c.handlers) {
		// 执行函数
		c.handlers[c.index](c)
	}
}
func (c *Context) Abort() {
	c.index = len(c.handlers)
}

type HandlerFun func(*Context)
type Engine struct {
	handlers []HandlerFun
}

func (e *Engine) Use(f HandlerFun) {
	// 将对应需要进行执行的处理函数都放到责任链中进行执行
	e.handlers = append(e.handlers, f)
}
func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	context := &Context{
		request:  r,
		w:        w,
		index:    -1,
		handlers: e.handlers,
	}
	context.Next()
}

func AuthMiddleware(c *Context) {
	fmt.Println("认证中间件")
}
func LogMiddleware(c *Context) {
	fmt.Println("日志中间件")
	c.Next()
}
func main() {
	// 将处理请求的对象使用链表串联起来，当有请求过来时，挨个儿传递请求，直到有对象处理请求未为止
	// 特别适用于将请求的发送者和接受者进行解耦
	r := &Engine{}
	r.Use(LogMiddleware)
	r.Use(AuthMiddleware)

	fmt.Println("web server on :8080")
	http.ListenAndServe(":8080", r)
}
