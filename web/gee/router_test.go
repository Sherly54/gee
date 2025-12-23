package gee

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

// 测试 parsePattern 函数
func TestParsePattern(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		expected []string
	}{
		{"simple path", "/user/list", []string{"user", "list"}},
		{"root path", "/", []string{}},
		{"path with params", "/user/:name", []string{"user", ":name"}},
		{"path with wildcard", "/static/*filepath", []string{"static", "*filepath"}},
		{"path with multiple params", "/user/:name/profile/:id", []string{"user", ":name", "profile", ":id"}},
		{"path with param and wildcard", "/user/:name/*filepath", []string{"user", ":name", "*filepath"}},
		{"path with empty segments", "/user//list/", []string{"user", "list"}},
		{"path with trailing wildcard", "/api/v1/*", []string{"api", "v1", "*"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePattern(tt.pattern)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parsePattern(%s) = %v, expected %v", tt.pattern, result, tt.expected)
			}
		})
	}
}

// 测试 extractParams 函数
func TestExtractParams(t *testing.T) {
	tests := []struct {
		name           string
		pattern        string
		searchPattern  string
		expectedParams map[string]string
	}{
		{"no params", "/user/list", "/user/list", map[string]string{}},
		{"single param", "/user/:name", "/user/alice", map[string]string{"name": "alice"}},
		{"multiple params", "/user/:name/profile/:id", "/user/alice/profile/123", map[string]string{"name": "alice", "id": "123"}},
		{"wildcard param", "/static/*filepath", "/static/css/main.css", map[string]string{"filepath": "css/main.css"}},
		{"wildcard with prefix", "/api/v1/*path", "/api/v1/users/123", map[string]string{"path": "users/123"}},
		{"param and wildcard", "/user/:name/*filepath", "/user/alice/docs/readme.md", map[string]string{"name": "alice", "filepath": "docs/readme.md"}},
		{"empty wildcard", "/static/*", "/static/", map[string]string{}},
		{"wildcard only", "/*", "/anything/here", map[string]string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractParams(tt.pattern, tt.searchPattern)
			if !reflect.DeepEqual(result, tt.expectedParams) {
				t.Errorf("extractParams(%s, %s) = %v, expected %v", tt.pattern, tt.searchPattern, result, tt.expectedParams)
			}
		})
	}
}

// 测试 Router 的 addRoute 和 getRoute 功能
func TestRouter_AddAndGetRoute(t *testing.T) {
	router := newRouter()

	// 添加路由
	router.addRoute("GET", "/user/list", nil)
	router.addRoute("GET", "/user/:name", nil)
	router.addRoute("POST", "/user/create", nil)
	router.addRoute("GET", "/static/*filepath", nil)

	tests := []struct {
		name        string
		method      string
		path        string
		expectedOk  bool
		expectedPat string
	}{
		{"exact match", "GET", "/user/list", true, "/user/list"},
		{"param match", "GET", "/user/alice", true, "/user/:name"},
		{"wildcard match", "GET", "/static/css/main.css", true, "/static/*filepath"},
		{"no match method", "POST", "/user/list", false, ""},
		{"no match path", "GET", "/post/123", false, ""},
		{"root path", "GET", "/", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, params := router.getRoute(tt.method, tt.path)
			if tt.expectedOk {
				if node == nil {
					t.Errorf("getRoute(%s, %s) returned nil node, expected non-nil", tt.method, tt.path)
				} else if node.pattern != tt.expectedPat {
					t.Errorf("getRoute(%s, %s) returned pattern %s, expected %s", tt.method, tt.path, node.pattern, tt.expectedPat)
				}
			} else {
				if node != nil {
					t.Errorf("getRoute(%s, %s) returned non-nil node, expected nil", tt.method, tt.path)
				}
			}
			// 验证参数提取
			if node != nil && node.pattern == "/user/:name" {
				expectedParams := map[string]string{"name": "alice"}
				if !reflect.DeepEqual(params, expectedParams) {
					t.Errorf("getRoute(%s, %s) returned params %v, expected %v", tt.method, tt.path, params, expectedParams)
				}
			}
			if node != nil && node.pattern == "/static/*filepath" {
				expectedParams := map[string]string{"filepath": "css/main.css"}
				if !reflect.DeepEqual(params, expectedParams) {
					t.Errorf("getRoute(%s, %s) returned params %v, expected %v", tt.method, tt.path, params, expectedParams)
				}
			}
		})
	}
}

// 测试 Router 的 getRoutes 功能
func TestRouter_GetRoutes(t *testing.T) {
	router := newRouter()

	// 添加路由
	router.addRoute("GET", "/user/list", nil)
	router.addRoute("GET", "/user/:name", nil)
	router.addRoute("POST", "/user/create", nil)
	router.addRoute("GET", "/static/*filepath", nil)

	// 测试 GET 方法的路由
	getRoutes := router.getRoutes("GET")
	if len(getRoutes) != 3 {
		t.Errorf("getRoutes(GET) returned %d routes, expected 3", len(getRoutes))
	}

	// 验证 GET 路由的 pattern（不关心顺序，使用 map 和 reflect.DeepEqual 进行高效对比）
	expectedPatterns := []string{"/user/list", "/user/:name", "/static/*filepath"}
	actualPatterns := make([]string, len(getRoutes))
	for i, route := range getRoutes {
		actualPatterns[i] = route.pattern
	}

	// 使用 map 统计每个 pattern 的数量
	expectedSet := make(map[string]int)
	actualSet := make(map[string]int)

	// 统计期望的 pattern 数量
	for _, pattern := range expectedPatterns {
		expectedSet[pattern]++
	}

	// 统计实际的 pattern 数量
	for _, pattern := range actualPatterns {
		actualSet[pattern]++
	}

	// 使用 reflect.DeepEqual 进行高效对比
	if !reflect.DeepEqual(expectedSet, actualSet) {
		t.Errorf("getRoutes(GET) patterns mismatch:\nexpected: %v\nactual: %v", expectedSet, actualSet)
	}

	// 测试 POST 方法的路由
	postRoutes := router.getRoutes("POST")
	if len(postRoutes) != 1 {
		t.Errorf("getRoutes(POST) returned %d routes, expected 1", len(postRoutes))
	}
	if postRoutes[0].pattern != "/user/create" {
		t.Errorf("getRoutes(POST) pattern = %s, expected /user/create", postRoutes[0].pattern)
	}

	// 测试不存在的方法
	noneRoutes := router.getRoutes("PUT")
	if noneRoutes != nil {
		t.Errorf("getRoutes(PUT) returned non-nil routes, expected nil")
	}
}

// 测试 Router 的 handle 功能
func TestRouter_Handle(t *testing.T) {
	router := newRouter()

	// 创建测试 handler
	var handled bool
	var capturedParams map[string]string
	testHandler := func(c *Context) {
		handled = true
		capturedParams = c.Params
		c.String(http.StatusOK, "handled")
	}

	// 添加路由
	router.addRoute("GET", "/user/:name", testHandler)

	// 创建测试 context
	req, _ := http.NewRequest("GET", "/user/alice", nil)
	w := httptest.NewRecorder()
	c := &Context{
		Writer: w,
		Req:    req,
		Path:   "/user/alice",
		Method: "GET",
		Params: make(map[string]string),
	}

	// 调用 handle
	router.handle(c)

	// 验证结果
	if !handled {
		t.Error("Handler was not called")
	}

	expectedParams := map[string]string{"name": "alice"}
	if !reflect.DeepEqual(capturedParams, expectedParams) {
		t.Errorf("Handler received params %v, expected %v", capturedParams, expectedParams)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Handler returned status code %d, expected %d", w.Code, http.StatusOK)
	}

	if w.Body.String() != "handled" {
		t.Errorf("Handler returned body '%s', expected 'handled'", w.Body.String())
	}
}

// 测试 Router 的 handle 功能 - 404 情况
func TestRouter_Handle_404(t *testing.T) {
	router := newRouter()

	// 创建测试 context
	req, _ := http.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()
	c := &Context{
		Writer: w,
		Req:    req,
		Path:   "/nonexistent",
		Method: "GET",
		Params: make(map[string]string),
	}

	// 调用 handle
	router.handle(c)

	// 验证 404 响应
	if w.Code != http.StatusNotFound {
		t.Errorf("Handler returned status code %d, expected %d", w.Code, http.StatusNotFound)
	}

	expectedBody := "404 NOT FOUND: GET, /nonexistent\n"
	if w.Body.String() != expectedBody {
		t.Errorf("Handler returned body '%s', expected '%s'", w.Body.String(), expectedBody)
	}
}
