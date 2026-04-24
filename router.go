package hush

import (
	"net/http"
	"strings"
)

// Router represents the Radix Tree based router.
// For simplicity in Phase 1, we will implement a basic prefix/parameter matcher,
// which can be optimized into a full Radix Tree later.
type Router struct {
	routes map[string]*node // HTTP Method -> Radix Tree Root
}

type node struct {
	path     string
	part     string
	children []*node
	isWild   bool
	handlers []HandlerFunc
}

func newRouter() *Router {
	return &Router{
		routes: make(map[string]*node),
	}
}

// insert adds a new route to the radix tree.
func (r *Router) insert(method, path string, handlers []HandlerFunc) {
	if _, ok := r.routes[method]; !ok {
		r.routes[method] = &node{}
	}
	
	parts := parsePath(path)
	root := r.routes[method]
	
	for _, part := range parts {
		child := root.matchChild(part)
		if child == nil {
			child = &node{
				part:   part,
				isWild: strings.HasPrefix(part, ":"),
			}
			root.children = append(root.children, child)
		}
		root = child
	}
	root.path = path
	root.handlers = handlers
}

// get finds the route and parses parameters.
func (r *Router) get(method, path string) (*node, map[string]string) {
	root, ok := r.routes[method]
	if !ok {
		return nil, nil
	}
	
	searchParts := parsePath(path)
	params := make(map[string]string)
	
	n := r.search(root, searchParts, params)
	if n != nil {
		return n, params
	}
	return nil, nil
}

func (r *Router) search(n *node, parts []string, params map[string]string) *node {
	if len(parts) == 0 {
		if n.path != "" {
			return n
		}
		return nil
	}
	
	part := parts[0]
	for _, child := range n.children {
		if child.part == part || child.isWild {
			if child.isWild {
				params[child.part[1:]] = part
			}
			result := r.search(child, parts[1:], params)
			if result != nil {
				return result
			}
		}
	}
	return nil
}

func (n *node) matchChild(part string) *node {
	for _, child := range n.children {
		if child.part == part {
			return child
		}
	}
	return nil
}

func parsePath(path string) []string {
	parts := strings.Split(path, "/")
	var result []string
	for _, part := range parts {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
