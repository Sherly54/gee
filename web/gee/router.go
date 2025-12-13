package gee

import (
	"log"
	"net/http"
)

type Router struct {
	handlers map[string]HandlerFunc
}

func newRouter() *Router {
	return &Router{handlers: make(map[string]HandlerFunc)}
}

func (r *Router) addRoute(method string, pattern string, handler HandlerFunc) {
	log.Printf("Route %4s - %4s", method, pattern)
	key := method + "-" + pattern
	r.handlers[key] = handler
}

func (r *Router) handle(c *Context) {
	key := c.Method + "-" + c.Path
	log.Printf("Handling %s", key)
	if handler, ok := r.handlers[key]; ok {
		handler(c)
	} else {
		c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
	}
}
