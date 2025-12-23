package gee

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// 测试节点基本结构
func TestNode_String(t *testing.T) {
	n := &node{pattern: "/user/:name", part: ":name"}
	expected := `node{pattern=/user/:name part=:name}`
	if n.String() != expected {
		t.Errorf("String() = %s, expected %s", n.String(), expected)
	}
}

// 测试正常插入（静态、参数、通配符）
func TestNode_Insert(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		parts   []string
		wantErr bool
	}{
		{"static", "/user/list", []string{"user", "list"}, false},
		{"param", "/user/:name", []string{"user", ":name"}, false},
		{"wildcard", "/static/*filepath", []string{"static", "*filepath"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.wantErr {
						t.Errorf("Insert() panicked unexpectedly: %v", r)
					}
				} else {
					if tt.wantErr {
						t.Error("Insert() did not panic, expected error")
					}
				}
			}()

			root := &node{}
			root.insert(tt.pattern, tt.parts, 0)

			if !hasPattern(root, tt.pattern, tt.parts) {
				t.Errorf("Not Found Pattern %s", tt.pattern)
			}
		})
	}
}

// 测试插入冲突（重复pattern、重复参数/通配符）
func TestNode_Insert_Conflict(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(root *node)
		wantErr bool
	}{
		{"duplicate pattern", func(root *node) {
			root.insert("/user/list", []string{"user", "list"}, 0)
			root.insert("/user/list", []string{"user", "list"}, 0)
		}, true},
		{"duplicate param", func(root *node) {
			root.insert("/user/:name", []string{"user", ":name"}, 0)
			root.insert("/user/:age", []string{"user", ":age"}, 0)
		}, true},
		{"duplicate wildcard", func(root *node) {
			root.insert("/static/*filepath", []string{"static", "*filepath"}, 0)
			root.insert("/static/*path", []string{"static", "*path"}, 0)
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := &node{}
			defer func() {
				if r := recover(); r != nil {
					if !tt.wantErr {
						t.Errorf("Insert() panicked unexpectedly: %v", r)
					}
				} else {
					if tt.wantErr {
						t.Error("Insert() did not panic, expected conflict error")
					}
				}
			}()
			tt.setup(root)
		})
	}
}

// 测试搜索功能（静态、参数、通配符、未找到）
func TestNode_Search(t *testing.T) {
	// 先构建测试Trie树
	root := &node{}
	// 插入测试节点
	root.insert("/user/list", []string{"user", "list"}, 0)
	root.insert("/user/:name", []string{"user", ":name"}, 0)
	root.insert("/static/*filepath", []string{"static", "*filepath"}, 0)
	root.insert("/", []string{}, 0) // 根节点

	tests := []struct {
		name    string
		parts   []string
		wantPat string
	}{
		{"root", []string{}, "/"},
		{"static match", []string{"user", "list"}, "/user/list"},
		{"param match", []string{"user", "alice"}, "/user/:name"},
		{"wildcard match", []string{"static", "css", "main.css"}, "/static/*filepath"},
		{"no match", []string{"post", "123"}, ""},
		{"partial match", []string{"user"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := root.search(tt.parts, 0)
			var gotPat string
			if node != nil {
				gotPat = node.pattern
			}
			if gotPat != tt.wantPat {
				t.Errorf("Search(%v) = %s, expected %s", tt.parts, gotPat, tt.wantPat)
			}
		})
	}
}

// 测试节点遍历功能
func TestNode_Travel(t *testing.T) {
	root := &node{}
	// 插入测试节点
	root.insert("/", []string{}, 0)
	root.insert("/user/list", []string{"user", "list"}, 0)
	root.insert("/user/:name", []string{"user", ":name"}, 0)
	root.insert("/static/*filepath", []string{"static", "*filepath"}, 0)

	// 期望的pattern列表（顺序：静态优先，参数次之，通配符最后）
	wantPatterns := []string{"/", "/user/list", "/user/:name", "/static/*filepath"}

	var list []*node
	root.travel(&list)

	// 提取遍历结果的pattern
	gotPatterns := make([]string, len(list))
	for i, n := range list {
		gotPatterns[i] = n.pattern
	}

	if !reflect.DeepEqual(gotPatterns, wantPatterns) {
		t.Errorf("Travel() got %v, expected %v", gotPatterns, wantPatterns)
	}
}

// 测试getOrCreateChild方法（创建不同类型子节点）
func TestNode_getOrCreateChild(t *testing.T) {
	root := &node{}

	// 测试创建静态子节点
	staticChild := root.getOrCreateChild("user")
	if staticChild.part != "user" || root.static["user"] != staticChild {
		t.Error("Failed to create static child node")
	}

	// 测试重复获取静态子节点
	staticChild2 := root.getOrCreateChild("user")
	if staticChild2 != staticChild {
		t.Error("Should return existing static child node")
	}

	// 测试创建参数子节点
	paramChild := root.getOrCreateChild(":name")
	if paramChild.part != ":name" || root.param != paramChild {
		t.Error("Failed to create param child node")
	}

	// 测试重复获取参数子节点
	paramChild2 := root.getOrCreateChild(":name")
	if paramChild2 != paramChild {
		t.Error("Should return existing param child node")
	}

	// 测试创建通配符子节点
	wildChild := root.getOrCreateChild("*filepath")
	if wildChild.part != "*filepath" || root.wildcard != wildChild {
		t.Error("Failed to create wildcard child node")
	}

	// 测试重复获取通配符子节点
	wildChild2 := root.getOrCreateChild("*filepath")
	if wildChild2 != wildChild {
		t.Error("Should return existing wildcard child node")
	}
}

// 测试参数子节点冲突
func TestNode_getOrCreateChild_ParamConflict(t *testing.T) {
	root := &node{}
	// 先创建一个参数子节点
	root.getOrCreateChild(":name")

	// 测试参数子节点冲突
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for conflicting param child, but none occurred")
		}
	}()

	root.getOrCreateChild(":age") // 不同参数名，应panic
}

// 测试getOrCreateChild方法中参数命名检查
func TestNode_getOrCreateChild_ParamNameValidation(t *testing.T) {
	root := &node{}

	// 测试空参数名（只有冒号）
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for empty param name, but none occurred")
		} else if !strings.Contains(fmt.Sprint(r), "Not Name For Param") {
			t.Errorf("Expected 'Not Name For Param' error, got: %v", r)
		}
	}()
	root.getOrCreateChild(":") // 空参数名，应panic
}

// 测试getOrCreateChild方法中通配符命名检查
func TestNode_getOrCreateChild_WildcardNameValidation(t *testing.T) {
	root := &node{}

	// 测试空通配符名（只有星号）
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for empty wildcard name, but none occurred")
		} else if !strings.Contains(fmt.Sprint(r), "Not Name For Wildcard") {
			t.Errorf("Expected 'Not Name For Wildcard' error, got: %v", r)
		}
	}()
	root.getOrCreateChild("*") // 空通配符名，应panic
}
