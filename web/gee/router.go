package gee

import (
	"log"
	"net/http"
	"strings"
)

type Router struct {
	handlers map[string]HandlerFunc
	root     map[string]*node
}

func newRouter() *Router {
	return &Router{handlers: make(map[string]HandlerFunc), root: make(map[string]*node)}
}

// Only one * is allowed
func parsePattern(pattern string) []string {
	vs := strings.Split(pattern, "/")

	parts := make([]string, 0)
	for _, item := range vs {
		if item != "" {
			parts = append(parts, item)
			if item[0] == '*' {
				break
			}
		}
	}
	return parts
}

func extractParams(pattern string, searchPattern string) map[string]string {
	params := make(map[string]string)

	parts := parsePattern(pattern)             // parts of node
	searchParts := parsePattern(searchPattern) // parts of searchUri

	// because of '*', len(parts) ≤ len(searchParts)
	for idx, part := range parts {
		switch part[0] {
		case ':':
			params[part[1:]] = searchParts[idx]
		case '*':
			if len(part) > 1 {
				params[part[1:]] = strings.Join(searchParts[idx:], "/")
			}
		}
	}

	return params
}

func (r *Router) addRoute(method string, pattern string, handler HandlerFunc) {
	log.Printf("Route %s - %s", method, pattern)

	parts := parsePattern(pattern)

	_, ok := r.root[method]
	if !ok {
		r.root[method] = &node{}
	}
	r.root[method].insert(pattern, parts, 0)

	key := method + "-" + pattern
	r.handlers[key] = handler
}

func (r *Router) getRoute(method string, path string) (*node, map[string]string) { // node, params
	root, ok := r.root[method]
	if !ok {
		return nil, nil
	}

	parts := parsePattern(path)
	n := root.search(parts, 0)
	if n == nil {
		return nil, nil
	}

	params := extractParams(n.pattern, path)
	return n, params
}

func (r *Router) getRoutes(method string) []*node {
	root, ok := r.root[method]
	if !ok {
		return nil
	}
	nodes := make([]*node, 0)
	root.travel(&nodes)
	return nodes
}

func (r *Router) handle(c *Context) {
	n, params := r.getRoute(c.Method, c.Path)
	if n != nil {
		c.Params = params
		key := c.Method + "-" + n.pattern
		r.handlers[key](c)
	} else {
		c.String(http.StatusNotFound, "404 NOT FOUND: %s, %s\n", c.Method, c.Path)
		return
	}
}
