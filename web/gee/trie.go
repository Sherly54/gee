package gee

import (
	"fmt"
	"strings"
)

// todo: recognize same param names
type node struct {
	// uri
	pattern string
	part    string

	// children
	static   map[string]*node
	param    *node
	wildcard *node
}

func hasPattern(root *node, pattern string, parts []string) bool {
	curr := root
	for _, part := range parts {
		var next *node
		switch part[0] {
		case ':':
			next = curr.param
		case '*':
			next = curr.wildcard
		default:
			next = curr.static[part]
		}
		if next == nil {
			return false
		}
		curr = next
	}
	return curr.pattern == pattern
}

func (n *node) String() string {
	return fmt.Sprintf("node{pattern=%s part=%s}", n.pattern, n.part)
}

func (n *node) insert(pattern string, parts []string, height int) {
	if height == len(parts) { // leaf, root is the node of height 0
		if n.pattern != "" {
			panic(fmt.Sprintf("[Trie] Pattern Conflict: Pattern: %s, Multiple Pattern For Node %s", pattern, n))
		}
		n.pattern = pattern
		return
	}

	part := parts[height]
	child := n.getOrCreateChild(part)
	child.insert(pattern, parts, height+1)
}

func (n *node) getOrCreateChild(part string) *node {
	switch part[0] { // part != nil
	case ':':
		if n.param != nil {
			if n.param.part != part {
				panic(fmt.Sprintf("[Trie] Pattern Conflict: Part: %s, Multiple Params Child For Node %s", part, n))
			}
			return n.param
		}
		if len(part) <= 1 {
			panic(fmt.Sprintf("[Trie] Not Name For Param, Part: %s", part))
		}
		n.param = &node{part: part}
		return n.param
	case '*': // node whose part starts with '*' must be leaf
		if n.wildcard != nil {
			if n.wildcard.part != part {
				panic(fmt.Sprintf("[Trie] Pattern Conflict: Part: %s, Multiple WildCard Child For Node %s", part, n))
			}
			return n.wildcard
		}
		if len(part) <= 1 {
			panic(fmt.Sprintf("[Trie] Not Name For Wildcard, Part: %s ", part))
		}
		n.wildcard = &node{part: part}
		return n.wildcard
	default:
		if n.static == nil {
			n.static = make(map[string]*node)
		}
		if child, ok := n.static[part]; ok {
			return child
		}
		n.static[part] = &node{part: part}
		return n.static[part]
	}
}

func (n *node) search(parts []string, height int) *node {
	if height == len(parts) {
		if n.pattern == "" {
			return nil
		}
		return n
	}

	if strings.HasPrefix(n.part, "*") {
		return n
	}

	part := parts[height]

	// order: static > param > wildcard
	if child, ok := n.static[part]; ok {
		if res := child.search(parts, height+1); res != nil {
			return res
		}
	}

	if n.param != nil {
		if res := n.param.search(parts, height+1); res != nil {
			return res
		}
	}

	if n.wildcard != nil { // wildcard must be leaf, no need to search
		return n.wildcard
	}

	return nil
}

func (n *node) travel(list *([]*node)) {
	if n.pattern != "" {
		*list = append(*list, n)
	}
	for _, child := range n.static {
		child.travel(list)
	}
	if n.param != nil {
		n.param.travel(list)
	}
	if n.wildcard != nil {
		n.wildcard.travel(list)
	}
}
